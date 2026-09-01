package main

import (
	"haskinproxy/internal/cache"
	"haskinproxy/internal/config"
	"haskinproxy/internal/handler"
	"haskinproxy/internal/upstream"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
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
	haClient := upstream.NewHAClient()
	appCache := cache.NewCache()
	cslHandler := handler.NewCSLHandler(haClient, appCache)
	webUIHandler := handler.NewWebUIHandler()

	// 2.1 Presence handshake with HRPAuth (bonjour), non-blocking
	if config.AppConfig.Presence.Enabled {
		announcePresence(haClient)
	}

	// 3. Setup router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WEBUI microservice integration (static routes take precedence over :username)
	// SDK JS relayed by HRPAuth to the WEBUI
	r.GET("/sdk/haskinproxy.js", webUIHandler.GetSDKJS)

	// CustomSkinLoader setup page embedded in the WEBUI Dashboard
	r.GET("/customskinloader", webUIHandler.GetCSLPage)

	// CSL Endpoints
	// /{username}.json
	r.GET("/:username", cslHandler.GetProfile)

	// /textures/{hash}
	r.GET("/textures/:hash", cslHandler.GetTexture)

	// 4. Start server
	addr := config.AppConfig.Server.ListenAddr
	log.Printf("HASkinProxy starting on %s", addr)
	if err := r.Run(addr); err != nil {
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
// the SDK URL the frontend loads, then registers the relay rule so the
// CustomSkinLoader page is reachable through the main service origin.
// Failures are only logged as warnings and never block the main process.
func announcePresence(cli *upstream.HAClient) {
	cfg := config.AppConfig.Presence
	go func() {
		req := upstream.PresenceRequest{
			Name:       cfg.Name,
			TTLSeconds: cfg.TTLSeconds,
			Scope: &upstream.PresenceScope{
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
		if err := cli.RegisterRelay(cfg.Name, []upstream.RelayRule{
			{Dest: "/customskinloader", Source: publicURL + "/customskinloader"},
		}); err != nil {
			log.Printf("WARN: relay rule registration with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("relay rule registered: /customskinloader -> %s/customskinloader", publicURL)
	}()
}
