package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/handler"
	"haskinproxy/internal/hrpauth"
	"haskinproxy/internal/router"
)

const configFileName = "config.yaml"

func main() {
	// 1. Load config (auto-generate if missing), located next to the
	// executable (like WinnerProxy) so it works regardless of CWD.
	cfgPath, err := configPath()
	if err != nil {
		log.Fatalf("locate executable: %v", err)
	}
	if err := config.LoadConfig(cfgPath); err != nil {
		log.Fatalf("Failed to load or generate config: %v", err)
	}

	// 1.5 Startup credential resolution: when the configured upstream
	// credentials are HRPAuth's built-in super client, they are exchanged
	// for this service's own low-privilege client via POST /oauth/register
	// (RFC 7591 protected registration) and the super secret is discarded
	// from memory. Must run before NewHAClient, which captures the
	// credentials from config.
	resolveUpstreamCredentials(cfgPath)

	// 2. Init components
	haClient := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	cslHandler := handler.NewCSLHandler(haClient, appCache)

	// 2.1 Presence handshake with HRPAuth (bonjour), non-blocking
	if config.AppConfig.Presence.Enabled {
		announcePresence(haClient)
	}

	// 2.2 Upload the compile-time SDK package to HRPAuth (if configured),
	// non-blocking. The WEBUI SDK handler aggregates it into the frontend
	// build; see HA-Contract sdk-package.md.
	if config.AppConfig.SDK.Enabled {
		uploadSDKPackage(haClient)
	}

	// 3. Setup router
	engine := router.New(cslHandler)

	// 4. Start server
	addr := config.AppConfig.Server.ListenAddr
	log.Printf("HASkinProxy starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// configPath returns the config file path next to the executable, so
// the proxy finds its config regardless of the current working
// directory (same approach as WinnerProxy).
func configPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), configFileName), nil
}

// resolveUpstreamCredentials implements the startup credential resolution
// for the upstream (HRPAuth) client credentials: when they turn out to be
// HRPAuth's built-in super client, they are exchanged for this service's
// own low-privilege client via POST /oauth/register (RFC 7591 protected
// registration) and the super secret is discarded from memory; credentials
// that are already a per-service client are verified and reused unchanged
// (idempotent path — no exchange, no write-back). Any failure aborts
// startup: super credentials must never reach business requests, and there
// is no fallback to running with unverified credentials.
func resolveUpstreamCredentials(cfgPath string) {
	up := &config.AppConfig.Upstream
	if up.ClientID == "" || up.ClientSecret == "" {
		return // credentials not configured; downstream calls fail as before
	}
	serviceName := config.AppConfig.Presence.Name
	if serviceName == "" {
		serviceName = config.DefaultConfig().Presence.Name
	}
	baseURL := strings.TrimRight(up.BaseURL, "/")
	superID, superSecret := up.ClientID, up.ClientSecret

	// The register endpoint only accepts presence-registered service
	// names, so the bonjour handshake goes first. It is a plain request:
	// no token may be attached before the credentials are resolved (the
	// super secret is used for the exchange call only). announcePresence
	// re-upserts the record asynchronously.
	if config.AppConfig.Presence.Enabled {
		if err := hrpauth.RegisterPresencePlain(baseURL, hrpauth.PresenceRequest{
			Name:       serviceName,
			TTLSeconds: config.AppConfig.Presence.TTLSeconds,
			Scope: &hrpauth.PresenceScope{
				Name:          "haskinproxy",
				FrontendAreas: []string{"webui-dash"},
			},
		}, nil); err != nil {
			log.Printf("Presence: bootstrap registration failed: %v (a super-credential exchange will be rejected until presence succeeds)", err)
		}
	}

	res, err := hrpauth.ResolveServiceCredentials(baseURL, serviceName, superID, superSecret, nil)
	if err != nil {
		log.Fatalf("credential resolution failed (refusing to start with unverified or super credentials): %v", err)
	}
	if res.Exchanged {
		log.Printf("credentials: super client detected and exchanged for per-service client %q; super credentials discarded from memory", res.ClientID)
		up.ClientID, up.ClientSecret = res.ClientID, res.ClientSecret
		applyResolvedCredentialsToConfigFile(cfgPath, superID, superSecret, res)
	} else {
		log.Printf("credentials: configured client %q confirmed as per-service client (no exchange needed)", up.ClientID)
	}
}

// applyResolvedCredentialsToConfigFile attempts the atomic config-file
// write-back after a successful exchange. Entrypoint-rendered deployments
// (CONFIG_TEMPLATE/RENDERED_CONFIG) are only warned about; a failed
// write-back is not fatal — the service keeps running with the in-memory
// credentials and the next startup exchanges again (the secret is stable
// server-side via HRPAuth get-or-create).
func applyResolvedCredentialsToConfigFile(cfgPath, superID, superSecret string, res *hrpauth.Resolution) {
	if config.IsRenderedDeployment() {
		log.Printf("WARN: config file is entrypoint-rendered (CONFIG_TEMPLATE/RENDERED_CONFIG); per-service credentials were NOT written back — set upstream.client_id/client_secret on the template/environment side instead; this deployment will exchange again on every start")
		return
	}
	n, err := config.RewriteClientCredentials(cfgPath, superID, superSecret, res.ClientID, res.ClientSecret, []string{"upstream"})
	if err != nil {
		log.Printf("ERROR: config file could not be rewritten with the per-service credentials (the service keeps running with the in-memory credentials): %v — next startup will perform the exchange again", err)
		return
	}
	log.Printf("credentials: config file rewritten with per-service client %q (sections replaced=%d, previous content kept at %s.bak)", res.ClientID, n, cfgPath)
}

// announcePresence performs the presence (bonjour) handshake with
// HRPAuth asynchronously. It registers HASkinProxy in HRPAuth's
// presence registry, declaring the WEBUI dashboard area it covers, then
// registers the relay rule so the CSL API (/csl/...) is reachable
// through the main service origin. The CustomSkinLoader frontend page is
// delivered separately as a compile-time SDK package (see uploadSDKPackage).
// Failures are only logged as warnings and never block the main process.
func announcePresence(cli *hrpauth.HAClient) {
	cfg := config.AppConfig.Presence
	go func() {
		publicURL := strings.TrimRight(config.AppConfig.Server.PublicURL, "/")

		req := hrpauth.PresenceRequest{
			Name:       cfg.Name,
			TTLSeconds: cfg.TTLSeconds,
			Scope: &hrpauth.PresenceScope{
				Name:          "haskinproxy",
				FrontendAreas: []string{"webui-dash"},
			},
		}
		if err := cli.RegisterPresence(req); err != nil {
			log.Printf("WARN: presence handshake with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("presence handshake ok: registered as %q", cfg.Name)

		if publicURL == "" {
			return
		}
		if err := cli.RegisterRelay(cfg.Name, []hrpauth.RelayRule{
			// CSL API（Minecraft 客户端）：主服务 origin + /csl/ 前缀，relay 剥离
			// 前缀后转发到本代理根路径（/:username、/textures/:hash）
			{Dest: "/csl", Source: publicURL},
		}); err != nil {
			log.Printf("WARN: relay rule registration with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("relay rules registered: /csl -> %s", publicURL)
	}()
}
