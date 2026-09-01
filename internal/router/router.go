// Package router wires the HTTP routes onto a gin engine, mirroring
// the internal/router package layout of WinnerProxy.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"haskinproxy/internal/handler"
)

// New builds the gin engine. Static routes (/sdk/..., /customskinloader)
// are registered before the /:username wildcard so they take precedence.
func New(csl *handler.CSLHandler, webui *handler.WebUIHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WEBUI microservice integration: SDK JS relayed by HRPAuth to the
	// WEBUI, and the CustomSkinLoader setup page embedded in the
	// Dashboard (reached through HRPAuth's relay).
	r.GET("/sdk/haskinproxy.js", webui.GetSDKJS)
	r.GET("/customskinloader", webui.GetCSLPage)

	// CSL endpoints.
	r.GET("/:username", csl.GetProfile)
	r.GET("/textures/:hash", csl.GetTexture)

	return r
}
