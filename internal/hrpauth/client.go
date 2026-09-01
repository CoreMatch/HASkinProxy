// Package hrpauth is the HRPAuth upstream client, mirroring the
// internal/hrpauth package layout of WinnerProxy.
package hrpauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"haskinproxy/config"
	"haskinproxy/internal/model"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type HAClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHAClient() *HAClient {
	return &HAClient{
		BaseURL: config.AppConfig.Upstream.BaseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(config.AppConfig.Upstream.Timeout) * time.Second,
		},
	}
}

// GetUUIDByUsername calls POST /api/profiles/minecraft
func (c *HAClient) GetUUIDByUsername(username string) (string, error) {
	url := c.BaseURL + "/api/profiles/minecraft"
	log.Printf("upstream request: POST %s (username=%s)", url, username)
	reqBody, _ := json.Marshal([]string{username})
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("upstream error: POST %s: %v", url, err)
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: POST %s -> %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var profiles []model.ProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return "", err
	}

	if len(profiles) == 0 {
		return "", fmt.Errorf("user not found")
	}

	return profiles[0].ID, nil
}

// GetProfileByUUID calls GET /sessionserver/session/minecraft/profile/:uuid
func (c *HAClient) GetProfileByUUID(uuid string) (*model.SessionProfileResponse, error) {
	url := c.BaseURL + "/sessionserver/session/minecraft/profile/" + uuid
	log.Printf("upstream request: GET %s", url)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		log.Printf("upstream error: GET %s: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: GET %s -> %d", url, resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("profile not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var profile model.SessionProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// DecodeTextures decodes the base64 textures property value
func DecodeTextures(property model.ProfileProperty) (*model.TexturePropertyValue, error) {
	if property.Name != "textures" {
		return nil, fmt.Errorf("not a textures property")
	}

	decoded, err := base64.StdEncoding.DecodeString(property.Value)
	if err != nil {
		return nil, err
	}

	var textures model.TexturePropertyValue
	if err := json.Unmarshal(decoded, &textures); err != nil {
		return nil, err
	}

	return &textures, nil
}

// FetchTexture fetches the raw texture bytes from /textures/:hash
func (c *HAClient) FetchTexture(hash string) ([]byte, http.Header, error) {
	url := c.BaseURL + "/textures/" + hash
	log.Printf("upstream request: GET %s", url)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		log.Printf("upstream error: GET %s: %v", url, err)
		return nil, nil, err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: GET %s -> %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return data, resp.Header, nil
}

// StatusResponse models GET /status data.backend (only the fields
// HASkinProxy uses are modeled).
type StatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Backend struct {
			Name    string `json:"name"`
			URL     string `json:"url"`
			Version string `json:"version"`
		} `json:"backend"`
	} `json:"data"`
}

// GetCallbackURL queries the main service's GET /status and returns
// data.backend.url (the main service callback URL / public origin).
// The sdk_url must be built on this origin (relayed URL), never on this
// proxy's internal address: clients (browser / WEBUI) are not in the
// same environment and cannot reach a localhost address.
//
//	200 with backend.url → url, nil
//	other status / error  → "", error
func (c *HAClient) GetCallbackURL() (string, error) {
	url := c.BaseURL + "/status"
	log.Printf("upstream request: GET %s", url)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		log.Printf("upstream error: GET %s: %v", url, err)
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: GET %s -> %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	var out StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Data.Backend.URL) == "" {
		return "", fmt.Errorf("backend.url is empty")
	}
	return out.Data.Backend.URL, nil
}

// PresenceScope is the scope declaration of a registered microservice.
// A non-empty FrontendAreas makes the service visible to the frontends
// whose areas overlap (see HA-Contract microservices.md).
type PresenceScope struct {
	Name          string   `json:"name"`
	FrontendAreas []string `json:"frontend_areas"`
}

// PresenceRequest is the body of the microservice presence handshake
// (POST /services/presence, the "bonjour" handshake). Only the fields
// HASkinProxy uses are modeled; optional contract fields (security_level,
// interacts_with) are omitted and stay unset.
type PresenceRequest struct {
	Name string `json:"name"`
	// TTLSeconds is the self-declared lifetime in seconds; <=0 or
	// omitted means the record never expires.
	TTLSeconds int `json:"ttl_seconds"`
	// Scope declares the frontend areas this service covers (e.g.
	// webui-dash) so the WEBUI can discover it.
	Scope *PresenceScope `json:"scope,omitempty"`
	// SDKURL points to the JS file that tells the frontend how to embed
	// this service; HRPAuth relays it unchanged via GET /services/sdk/:name.
	SDKURL string `json:"sdk_url,omitempty"`
}

// RegisterPresence performs the microservice presence (bonjour)
// handshake: it registers or heartbeats HASkinProxy in HRPAuth's
// presence registry. It is fire-and-forget from the caller's point of
// view; failures surface as an error and never stop the process.
//
//	200          → nil
//	network err  → error
//	other status → error
func (c *HAClient) RegisterPresence(req PresenceRequest) error {
	url := c.BaseURL + "/services/presence"
	log.Printf("upstream request: POST %s (name=%s)", url, req.Name)
	body, _ := json.Marshal(req)
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("upstream error: POST %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: POST %s -> %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	return nil
}

// RelayRule maps a public path on the main service (dest) to a
// microservice address (source). Requests hitting dest (and sub-paths)
// are forwarded to source + the remaining path (see HA-Contract
// microservices.md, "Relay Rules").
type RelayRule struct {
	Dest   string `json:"dest"`
	Source string `json:"source"`
}

// RegisterRelay registers relay rules with HRPAuth (POST /services/relay)
// so the frontend can reach this proxy through the main service origin.
// Requires the service to have completed the presence handshake first.
//
//	200          → nil
//	network err  → error
//	other status → error
func (c *HAClient) RegisterRelay(name string, relays []RelayRule) error {
	url := c.BaseURL + "/services/relay"
	log.Printf("upstream request: POST %s (name=%s, rules=%d)", url, name, len(relays))
	body, _ := json.Marshal(map[string]any{
		"name":   name,
		"relays": relays,
	})
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("upstream error: POST %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: POST %s -> %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	return nil
}
