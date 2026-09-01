package main

import (
	"haskinproxy/internal/cache"
	"haskinproxy/internal/config"
	"haskinproxy/internal/handler"
	"haskinproxy/internal/upstream"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load config (auto-generate if missing)
	if err := config.LoadConfig("config.yaml"); err != nil {
		log.Fatalf("Failed to load or generate config.yaml: %v", err)
	}

	// 2. Init components
	haClient := upstream.NewHAClient()
	appCache := cache.NewCache()
	cslHandler := handler.NewCSLHandler(haClient, appCache)

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

// announcePresence performs the presence (bonjour) handshake with
// HRPAuth asynchronously. It registers HASkinProxy in HRPAuth's
// presence registry; a failure is only logged as a warning and never
// blocks or stops the main process.
func announcePresence(cli *upstream.HAClient) {
	cfg := config.AppConfig.Presence
	go func() {
		if err := cli.RegisterPresence(upstream.PresenceRequest{
			Name:       cfg.Name,
			TTLSeconds: cfg.TTLSeconds,
		}); err != nil {
			log.Printf("WARN: presence handshake with HRPAuth failed (proxy continues running): %v", err)
			return
		}
		log.Printf("presence handshake ok: registered as %q", cfg.Name)
	}()
}
