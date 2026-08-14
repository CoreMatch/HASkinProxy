package handler

import (
	"haskinproxy/internal/cache"
	"haskinproxy/internal/config"
	"haskinproxy/internal/model"
	"haskinproxy/internal/upstream"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CSLHandler struct {
	HAClient *upstream.HAClient
	Cache    *cache.Cache
}

func NewCSLHandler(client *upstream.HAClient, c *cache.Cache) *CSLHandler {
	return &CSLHandler{
		HAClient: client,
		Cache:    c,
	}
}

// GetProfile handles GET /{username}.json
func (h *CSLHandler) GetProfile(c *gin.Context) {
	usernameJSON := c.Param("username")
	if !strings.HasSuffix(usernameJSON, ".json") {
		c.Status(http.StatusNotFound)
		return
	}
	username := strings.TrimSuffix(usernameJSON, ".json")

	// Try cache
	var cachedCSL model.CSLProfile
	if found, _ := h.Cache.Get("profile:"+username, &cachedCSL); found {
		c.JSON(http.StatusOK, cachedCSL)
		return
	}

	// 1. Get UUID
	uuid, err := h.HAClient.GetUUIDByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 2. Get Profile
	profile, err := h.HAClient.GetProfileByUUID(uuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 3. Convert to CSL
	csl := model.CSLProfile{
		Username: profile.Name,
		Textures: make(map[string]string),
	}

	for _, prop := range profile.Properties {
		if prop.Name == "textures" {
			texVal, err := upstream.DecodeTextures(prop)
			if err != nil {
				continue
			}

			if texVal.Textures.SKIN != nil {
				hash := extractHash(texVal.Textures.SKIN.URL)
				modelType := "default"
				if texVal.Textures.SKIN.Metadata != nil {
					if m, ok := texVal.Textures.SKIN.Metadata["model"]; ok {
						modelType = m
					}
				}
				csl.Textures[modelType] = hash
			}
			if texVal.Textures.CAPE != nil {
				csl.Textures["cape"] = extractHash(texVal.Textures.CAPE.URL)
			}
			if texVal.Textures.ELYTRA != nil {
				csl.Textures["elytra"] = extractHash(texVal.Textures.ELYTRA.URL)
			}
		}
	}

	// Set cache
	h.Cache.Set("profile:"+username, csl, config.AppConfig.Cache.ProfileTTL)

	c.JSON(http.StatusOK, csl)
}

// GetTexture handles GET /textures/{hash}
func (h *CSLHandler) GetTexture(c *gin.Context) {
	hash := c.Param("hash")

	// Try cache
	if data, found, _ := h.Cache.GetRaw("tex:" + hash); found {
		c.Header("Content-Type", "image/png")
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "image/png", data)
		return
	}

	data, header, err := h.HAClient.FetchTexture(hash)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// Forward key headers
	contentType := header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	c.Header("Content-Type", contentType)

	if lm := header.Get("Last-Modified"); lm != "" {
		c.Header("Last-Modified", lm)
	}
	if cc := header.Get("Cache-Control"); cc != "" {
		c.Header("Cache-Control", cc)
	}

	// Set cache
	h.Cache.SetRaw("tex:"+hash, data, config.AppConfig.Cache.TextureTTL)

	c.Data(http.StatusOK, contentType, data)
}

func extractHash(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
