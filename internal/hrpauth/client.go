// Package hrpauth is the HRPAuth upstream client, mirroring the
// internal/hrpauth package layout of WinnerProxy.
package hrpauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"haskinproxy/config"
	"haskinproxy/internal/model"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type HAClient struct {
	BaseURL      string
	HTTPClient   *http.Client
	ClientID     string
	ClientSecret string
}

func NewHAClient() *HAClient {
	return &HAClient{
		BaseURL: config.AppConfig.Upstream.BaseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(config.AppConfig.Upstream.Timeout) * time.Second,
		},
		ClientID:     config.AppConfig.Upstream.ClientID,
		ClientSecret: config.AppConfig.Upstream.ClientSecret,
	}
}

func (c *HAClient) do(req *http.Request) (*http.Response, error) {
	// If ClientID and ClientSecret are set, we assume ClientSecret is a
	// static OAuth2 token (as per "oauth2 client id/token" instructions).
	// We only set it if the caller hasn't already provided an Authorization
	// header (e.g. for user-initiated proxy requests).
	if c.ClientID != "" && c.ClientSecret != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.ClientSecret)
		req.Header.Set("X-Client-ID", c.ClientID)
	}
	return c.HTTPClient.Do(req)
}

// GetUUIDByUsername calls POST /api/profiles/minecraft
func (c *HAClient) GetUUIDByUsername(username string) (string, error) {
	url := c.BaseURL + "/api/profiles/minecraft"
	log.Printf("upstream request: POST %s (username=%s)", url, username)
	reqBody, _ := json.Marshal([]string{username})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
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
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
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

// DeleteTexture calls POST /texture/delete on the upstream. The body
// and Authorization header are forwarded unchanged from the caller
// (HASkinProxy itself does not authenticate the deletion). Returns the
// raw response body, status code and upstream response headers.
//
// The caller (handler) is expected to map client-side identifiers (e.g.
// username) to upstream identifiers (profile_id) before invoking this
// method, since the upstream body carries upstream-shaped fields.
//
//	2xx          → body, status, header, nil
//	other status → body, status, header, nil  (upstream decided the failure)
//	network err  → nil, 0, nil, err
func (c *HAClient) DeleteTexture(body []byte, authHeader string) ([]byte, int, http.Header, error) {
	url := c.BaseURL + "/texture/delete"
	log.Printf("upstream request: POST %s", url)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := c.do(req)
	if err != nil {
		log.Printf("upstream error: POST %s: %v", url, err)
		return nil, 0, nil, err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: POST %s -> %d", url, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}
	return respBody, resp.StatusCode, resp.Header, nil
}

// FetchTexture fetches the raw texture bytes from /textures/:hash
func (c *HAClient) FetchTexture(hash string) ([]byte, http.Header, error) {
	url := c.BaseURL + "/textures/" + hash
	log.Printf("upstream request: GET %s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.do(req)
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
// interacts_with) are omitted and stay unset. The legacy sdk_url field
// is gone: SDKs are now delivered as source packages aggregated into the
// frontend at compile time (see HA-Contract sdk-package.md).
type PresenceRequest struct {
	Name string `json:"name"`
	// TTLSeconds is the self-declared lifetime in seconds; <=0 or
	// omitted means the record never expires.
	TTLSeconds int `json:"ttl_seconds"`
	// Scope declares the frontend areas this service covers (e.g.
	// webui-dash) so the WEBUI can discover it.
	Scope *PresenceScope `json:"scope,omitempty"`
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
	hReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	hReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(hReq)
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

// ErrSDKPackageConflict is returned by UploadSDKPackage when a package
// with the same name already exists on HRPAuth (409 sdk_package_conflict).
// Replacing requires DELETE first (see HA-Contract sdk-package.md).
var ErrSDKPackageConflict = errors.New("sdk package with the same name already exists")

// UploadSDKPackage uploads a compile-time SDK package archive to HRPAuth
// (POST /services/sdk-packages, multipart/form-data field "package").
// Authentication reuses the client's existing Bearer ClientSecret
// credentials, which resolve to Ops Level 2 on the main service.
//
//	2xx  (201)        → nil
//	409               → ErrSDKPackageConflict
//	other status      → error
//	network error     → error
func (c *HAClient) UploadSDKPackage(data []byte, filename string) error {
	url := c.BaseURL + "/services/sdk-packages"

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("package", filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.do(req)
	if err != nil {
		log.Printf("upstream error: POST %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: POST %s -> %d", url, resp.StatusCode)

	switch {
	case resp.StatusCode == http.StatusConflict:
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrSDKPackageConflict
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// DeleteSDKPackage removes a previously uploaded SDK package
// (DELETE /services/sdk-packages/:name). A 404 is treated as success since
// the desired state is already reached.
func (c *HAClient) DeleteSDKPackage(name string) error {
	url := c.BaseURL + "/services/sdk-packages/" + name
	log.Printf("upstream request: DELETE %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		log.Printf("upstream error: DELETE %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("upstream response: DELETE %s -> %d", url, resp.StatusCode)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
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
