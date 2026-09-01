package upstream

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"haskinproxy/internal/config"
	"haskinproxy/internal/model"
	"io"
	"net/http"
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
	reqBody, _ := json.Marshal([]string{username})
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/profiles/minecraft", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
	resp, err := c.HTTPClient.Get(c.BaseURL + "/sessionserver/session/minecraft/profile/" + uuid)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	resp, err := c.HTTPClient.Get(c.BaseURL + "/textures/" + hash)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return data, resp.Header, nil
}

// PresenceRequest is the body of the microservice presence handshake
// (POST /services/presence, the "bonjour" handshake). Only the fields
// HASkinProxy uses are modeled; optional contract fields (scope,
// sdk_url, security_level, interacts_with) are omitted and stay unset.
type PresenceRequest struct {
	Name string `json:"name"`
	// TTLSeconds is the self-declared lifetime in seconds; <=0 or
	// omitted means the record never expires.
	TTLSeconds int `json:"ttl_seconds"`
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
	body, _ := json.Marshal(req)
	resp, err := c.HTTPClient.Post(c.BaseURL+"/services/presence", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	return nil
}
