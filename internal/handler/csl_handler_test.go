package handler

import (
	"encoding/json"
	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/hrpauth"
	"haskinproxy/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSLHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Mock HA Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/yggdrasil-api/api/profiles/minecraft":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]model.ProfileResponse{{ID: "test-uuid", Name: "TestUser"}})
		case "/yggdrasil-api/sessionserver/session/minecraft/profile/test-uuid":
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

func TestCSLHandler_DeleteTexture(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Mock HA Server: the current upstream route is
	// DELETE /texture/delete/:uuid/:type under the /yggdrasil-api relay.
	var gotMethod, gotPath, gotAuth string
	upstreamBody := `{"success":true,"message":"Texture deleted successfully"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if r.Method == http.MethodDelete && r.URL.Path == "/yggdrasil-api/texture/delete/test-uuid/skin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(upstreamBody))
			return
		}
		w.WriteHeader(http.StatusNotFound)
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

	// Seed the local cache so the success path must evict it.
	appCache.Set("profile:TestUser", model.CSLProfile{
		Username: "TestUser",
		Textures: map[string]string{"default": "abc123"},
	}, 60)
	appCache.SetRaw("tex:abc123", []byte("png-bytes"), 60)

	r := gin.Default()
	r.POST("/:username/texture/delete", h.DeleteTexture)

	// 3. Test Request
	reqBody := `{"profile_id":"test-uuid","texture_type":"skin","remember_token":"legacy","uid":42}`
	req, _ := http.NewRequest("POST", "/TestUser/texture/delete", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer user-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 4. Assertions: upstream shape, auth forwarding, response passthrough,
	// cache eviction.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/yggdrasil-api/texture/delete/test-uuid/skin", gotPath)
	assert.Equal(t, "Bearer user-token", gotAuth)
	assert.JSONEq(t, upstreamBody, w.Body.String())

	var cached model.CSLProfile
	found, _ := appCache.Get("profile:TestUser", &cached)
	assert.False(t, found, "profile cache entry must be evicted on success")
	_, texFound, _ := appCache.GetRaw("tex:abc123")
	assert.False(t, texFound, "texture cache entries must be evicted on success")
}

func TestCSLHandler_DeleteTexture_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamHit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config.AppConfig.Upstream.BaseURL = server.URL
	config.AppConfig.Upstream.Timeout = 5
	config.AppConfig.Cache.ProfileTTL = 60
	config.AppConfig.Cache.MaxSizeMB = 10

	client := hrpauth.NewHAClient()
	h := NewCSLHandler(client, cache.NewCache())

	r := gin.Default()
	r.POST("/:username/texture/delete", h.DeleteTexture)

	for _, body := range []string{`{}`, `{"profile_id":"test-uuid"}`, `{"texture_type":"skin"}`, `not-json`} {
		req, _ := http.NewRequest("POST", "/TestUser/texture/delete", strings.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, "body=%q", body)
	}
	assert.False(t, upstreamHit, "upstream must not be called when body fields are missing")
}

func TestCSLHandler_DeleteTexture_UpstreamStatusForwarded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"message":"Forbidden","code":"profile_access_denied"}`))
	}))
	defer server.Close()

	config.AppConfig.Upstream.BaseURL = server.URL
	config.AppConfig.Upstream.Timeout = 5
	config.AppConfig.Cache.ProfileTTL = 60
	config.AppConfig.Cache.MaxSizeMB = 10

	client := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	h := NewCSLHandler(client, appCache)
	appCache.Set("profile:TestUser", model.CSLProfile{
		Username: "TestUser",
		Textures: map[string]string{"default": "abc123"},
	}, 60)

	r := gin.Default()
	r.POST("/:username/texture/delete", h.DeleteTexture)

	req, _ := http.NewRequest("POST", "/TestUser/texture/delete",
		strings.NewReader(`{"profile_id":"test-uuid","texture_type":"skin"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"success":false,"message":"Forbidden","code":"profile_access_denied"}`, w.Body.String())

	var cached model.CSLProfile
	found, _ := appCache.Get("profile:TestUser", &cached)
	assert.True(t, found, "cache must not be evicted when the upstream rejects the delete")
}
