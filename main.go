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

	// 2. Init components
	haClient := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	cslHandler := handler.NewCSLHandler(haClient, appCache)
	webUIHandler := handler.NewWebUIHandler()

	// 2.1 Presence handshake with HRPAuth (bonjour), non-blocking
	if config.AppConfig.Presence.Enabled {
		announcePresence(haClient)
	}

	// 3. Setup router
	engine := router.New(cslHandler, webUIHandler)

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

// announcePresence performs the presence (bonjour) handshake with
// HRPAuth asynchronously. It registers HASkinProxy in HRPAuth's
// presence registry, declaring the WEBUI dashboard area it covers and
// the SDK URL the frontend loads, then registers the relay rules so the
// CustomSkinLoader page and the CSL API are reachable through the main
// service origin (/customskinloader, /csl/...).
// Failures are only logged as warnings and never block the main process.
func announcePresence(cli *hrpauth.HAClient) {
	cfg := config.AppConfig.Presence
	go func() {
		req := hrpauth.PresenceRequest{
			Name:       cfg.Name,
			TTLSeconds: cfg.TTLSeconds,
			Scope: &hrpauth.PresenceScope{
				Name:          "haskinproxy",
				FrontendAreas: []string{"webui-dash"},
			},
		}
		publicURL := strings.TrimRight(config.AppConfig.Server.PublicURL, "/")
		if publicURL != "" {
			req.SDKURL = publicURL + "/sdk/haskinproxy.js"
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
			{Dest: "/customskinloader", Source: publicURL + "/customskinloader"},
			// The CSL API lives at this proxy's root ({username}.json,
			// textures/{hash}); /csl/{path} on the main service maps to
			// {publicURL}/{path} (prefix concatenation).
			{Dest: "/csl", Source: publicURL},
		}); err != nil {
			log.Printf("WARN: relay rule registration with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("relay rules registered: /customskinloader, /csl -> %s", publicURL)
	}()
}
