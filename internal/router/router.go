// Package router wires the HTTP routes onto a gin engine, mirroring
// the internal/router package layout of WinnerProxy.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"haskinproxy/internal/handler"
)

// New builds the gin engine. Static routes (/health, /textures/...) are
// registered before the /:username wildcard so they take precedence.
//
// Note: the WEBUI-facing surface (/customskinloader setup page) is now
// shipped as a compile-time SDK package (see sdk/ in this repo and
// HA-Contract sdk-package.md); no runtime route is served for it anymore.
func New(csl *handler.CSLHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// CSL endpoints.
	r.GET("/:username", csl.GetProfile)
	r.GET("/textures/:hash", csl.GetTexture)
	r.POST("/:username/texture/delete", csl.DeleteTexture)

	return r
}
