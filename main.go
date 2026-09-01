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
// SDK file, the CustomSkinLoader page and the CSL API are reachable
// through the main service origin (/sdk/haskinproxy.js, /customskinloader,
// /csl/...).
// Failures are only logged as warnings and never block the main process.
func announcePresence(cli *hrpauth.HAClient) {
	cfg := config.AppConfig.Presence
	go func() {
		publicURL := strings.TrimRight(config.AppConfig.Server.PublicURL, "/")

		// sdk_url must be built on the main service's callback origin
		// ({callback.url}): that is a relayed, publicly reachable address.
		// Never advertise this proxy's internal address (e.g.
		// localhost:2702) as sdk_url — clients (browser / WEBUI) are not
		// in the same environment and cannot reach it.
		mainBase := publicURL
		if cb, err := cli.GetCallbackURL(); err == nil && cb != "" {
			mainBase = strings.TrimRight(cb, "/")
		} else {
			log.Printf("WARN: could not resolve main service callback url from /status (%v); sdk_url falls back to public_url", err)
		}

		req := hrpauth.PresenceRequest{
			Name:       cfg.Name,
			TTLSeconds: cfg.TTLSeconds,
			Scope: &hrpauth.PresenceScope{
				Name:          "haskinproxy",
				FrontendAreas: []string{"webui-dash"},
			},
		}
		if mainBase != "" {
			req.SDKURL = mainBase + "/sdk/haskinproxy.js"
		}
		if err := cli.RegisterPresence(req); err != nil {
			log.Printf("WARN: presence handshake with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("presence handshake ok: registered as %q (sdk_url=%s)", cfg.Name, req.SDKURL)

		if publicURL == "" {
			return
		}
		if err := cli.RegisterRelay(cfg.Name, []hrpauth.RelayRule{
			// SDK 文件本身也经主服务 relay：HRPAuth 服务端抓取 sdk_url
			//（{callback.url}/sdk/haskinproxy.js）时通过本规则回源到本代理
			{Dest: "/sdk/haskinproxy.js", Source: publicURL + "/sdk/haskinproxy.js"},
			// iframe 页面（浏览器）：主服务 origin + /customskinloader -> 本代理配置页
			{Dest: "/customskinloader", Source: publicURL + "/customskinloader"},
			// CSL API（Minecraft 客户端）：主服务 origin + /csl/ 前缀，relay 剥离
			// 前缀后转发到本代理根路径（/users/:username、/textures/:hash）
			{Dest: "/csl", Source: publicURL},
		}); err != nil {
			log.Printf("WARN: relay rule registration with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("relay rules registered: /sdk/haskinproxy.js, /customskinloader, /csl -> %s", publicURL)
	}()
}
