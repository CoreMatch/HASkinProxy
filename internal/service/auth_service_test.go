package service

import (
	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/hrpauth"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthService_SendResetCode_Privacy(t *testing.T) {
	// Setup mock upstream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("email") == "nonexistent@example.com" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config.AppConfig.Upstream.BaseURL = server.URL
	haClient := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	s := NewAuthService(appCache, haClient)

	// Test non-existent email
	err := s.SendResetCode("nonexistent@example.com")
	assert.NoError(t, err, "Should return success even if email does not exist")

	// Verify no code is stored in cache for non-existent email
	_, found, _ := appCache.GetRaw("reset_code:nonexistent@example.com")
	assert.False(t, found, "No code should be stored for non-existent email")
}

func TestAuthService_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config.AppConfig.Upstream.BaseURL = server.URL
	haClient := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	s := NewAuthService(appCache, haClient)

	email := "test@example.com"

	// 1st request
	err := s.SendResetCode(email)
	assert.NoError(t, err)

	// 2nd request within a minute
	err = s.SendResetCode(email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wait a minute")

	// Force clear minute limit but simulate 10 requests for hour limit
	for i := 0; i < 9; i++ {
		s.updateRateLimit(email)
	}
	appCache.Delete("rate:min:" + email)

	// 11th request within an hour
	err = s.SendResetCode(email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many requests")
}
