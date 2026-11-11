package service

import (
	"encoding/json"
	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/hrpauth"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newUpstreamMock returns an httptest server emulating the HRPAuth core
// POST /internal/user-exists endpoint. Emails in unknownEmails are reported
// as not existing; every other email is reported as existing.
func newUpstreamMock(unknownEmails ...string) *httptest.Server {
	unknown := map[string]bool{}
	for _, e := range unknownEmails {
		unknown[e] = true
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		exists := !unknown[req.Email]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"exists": exists},
		})
	}))
}

// configureTestEnv points the global config at the mock upstream and at an
// SMTP target that fails fast (no DNS lookups) so the background sendEmail
// goroutine never hangs a test.
func configureTestEnv(upstreamURL string) {
	config.AppConfig = config.DefaultConfig()
	config.AppConfig.Upstream.BaseURL = upstreamURL
	config.AppConfig.SMTP.Host = "127.0.0.1"
	config.AppConfig.SMTP.Port = 1 // nothing listens; sendEmail fails fast
}

func TestAuthService_SendResetCode_Privacy(t *testing.T) {
	server := newUpstreamMock("nonexistent@example.com")
	defer server.Close()

	configureTestEnv(server.URL)
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

func setupAuthTestEnv(t *testing.T) (*AuthService, *cache.Cache) {
	server := newUpstreamMock()
	t.Cleanup(server.Close)
	configureTestEnv(server.URL)
	haClient := hrpauth.NewHAClient()
	appCache := cache.NewCache()
	return NewAuthService(appCache, haClient), appCache
}

func TestAuthService_RateLimit(t *testing.T) {
	s, appCache := setupAuthTestEnv(t)

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

func TestAuthService_VerifyResetCode(t *testing.T) {
	s, appCache := setupAuthTestEnv(t)

	email := "verify@example.com"
	err := s.SendResetCode(email)
	assert.NoError(t, err)

	// Retrieve the stored code directly to verify against it.
	var stored string
	found, _ := appCache.Get("reset_code:"+email, &stored)
	assert.True(t, found, "a code should be stored for an existing email")

	wrong := "0000"
	if stored == wrong {
		wrong = "0001"
	}
	assert.True(t, s.VerifyResetCode(email, stored), "correct code verifies")
	assert.False(t, s.VerifyResetCode(email, wrong), "wrong code rejected")
	assert.False(t, s.VerifyResetCode("unknown@example.com", stored), "unknown email rejected")
}
