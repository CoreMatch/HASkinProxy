package handler

import (
	"encoding/json"
	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/hrpauth"
	"haskinproxy/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSLHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Mock HA Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/profiles/minecraft":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]model.ProfileResponse{{ID: "test-uuid", Name: "TestUser"}})
		case "/sessionserver/session/minecraft/profile/test-uuid":
			w.Header().Set("Content-Type", "application/json")
			profile := model.SessionProfileResponse{
				ID:   "test-uuid",
				Name: "TestUser",
				Properties: []model.ProfileProperty{
					{
						Name:  "textures",
						Value: "eyJ0ZXh0dXJlcyI6eyJTS0lOIjp7InVybCI6Imh0dHA6Ly9sb2NhbGhvc3QvdGV4dHVyZXMvYWJjMTIzIn19fQ==", // {"textures":{"SKIN":{"url":"http://localhost/textures/abc123"}}}
					},
				},
			}
			json.NewEncoder(w).Encode(profile)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 2. Setup Handler
	config.AppConfig.Upstream.BaseURL = server.URL
	config.AppConfig.Upstream.Timeout = 5
	config.AppConfig.Cache.ProfileTTL = 60
	config.AppConfig.Cache.MaxSizeMB = 10

	client := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	h := NewCSLHandler(client, appCache)

	r := gin.Default()
	r.GET("/:username", h.GetProfile)

	// 3. Test Request
	req, _ := http.NewRequest("GET", "/TestUser.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 4. Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.CSLProfile
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "TestUser", resp.Username)
	assert.Equal(t, "abc123", resp.Textures["default"])
}
