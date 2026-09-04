package handler

import (
	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/hrpauth"
	"haskinproxy/internal/model"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CSLHandler struct {
	HAClient *hrpauth.HAClient
	Cache    *cache.Cache
}

func NewCSLHandler(client *hrpauth.HAClient, c *cache.Cache) *CSLHandler {
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
			texVal, err := hrpauth.DecodeTextures(prop)
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

// DeleteTexture handles POST /csl/:username/texture/delete. It forwards
// the request body unchanged to the upstream POST /texture/delete,
// along with the caller's Authorization header (HASkinProxy does not
// authenticate deletions itself). The :username path segment is used
// only to identify which local cache entries to evict on success.
//
// Request body: whatever the upstream POST /texture/delete accepts
// (remember_token, profile_id, texture_type, auth_type, uid, email).
// Response body: the upstream response forwarded as-is, with the
// upstream status code.
func (h *CSLHandler) DeleteTexture(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	// Pre-flight: capture the existing texture hashes from the local
	// profile cache so we can evict the matching tex: entries on
	// success. If the profile is not cached, only the profile entry is
	// evicted (tex entries are left to expire via TTL).
	var texHashes []string
	var cachedCSL model.CSLProfile
	if found, _ := h.Cache.Get("profile:"+username, &cachedCSL); found {
		for _, hash := range cachedCSL.Textures {
			if hash != "" {
				texHashes = append(texHashes, hash)
			}
		}
	}

	respBody, status, _, err := h.HAClient.DeleteTexture(body, c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Evict local cache on success. Use upstream status as ground
	// truth; the upstream response body is forwarded regardless.
	if status >= 200 && status < 300 {
		h.Cache.Delete("profile:" + username)
		for _, hash := range texHashes {
			h.Cache.Delete("tex:" + hash)
		}
	}

	// Forward upstream response: status + Content-Type (default JSON) +
	// raw body.
	c.Header("Content-Type", "application/json")
	c.Data(status, "application/json", respBody)
}
