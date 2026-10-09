// Package config loads HASkinProxy's runtime configuration from a YAML
// file, mirroring the layout of WinnerProxy's top-level config package.
package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		ListenAddr string `yaml:"listen_addr"`
		// PublicURL is the externally reachable base URL of this proxy
		// (e.g. http://localhost:2702). It is announced to HRPAuth as the
		// relay source so the frontend can reach the CustomSkinLoader
		// setup page through the main service origin.
		PublicURL string `yaml:"public_url"`
	} `yaml:"server"`
	Upstream struct {
		// BaseURL is the HRPAuth main service base URL. Yggdrasil API
		// requests are routed through HA's /yggdrasil-api relay prefix.
		BaseURL string `yaml:"base_url"`
		Timeout int    `yaml:"timeout"` // in seconds
		// Client credentials are exchanged for an OAuth2 service token via
		// POST /oauth/token (grant_type=client_credentials).
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	} `yaml:"upstream"`
	Cache struct {
		ProfileTTL int `yaml:"profile_ttl"` // in seconds
		TextureTTL int `yaml:"texture_ttl"` // in seconds
		MaxSizeMB  int `yaml:"max_size_mb"`
	} `yaml:"cache"`
	Presence PresenceConfig `yaml:"presence"`
	SDK      SDKConfig      `yaml:"sdk"`
}

// SDKConfig controls the compile-time SDK package upload. On startup the
// proxy packs the embedded sdk/ source tree (manifest.json + React page)
// into a tar.gz archive and uploads it to HRPAuth (POST
// /services/sdk-packages); the WebUI SDK handler aggregates it into the
// frontend build (see HA-Contract sdk-package.md). A failed upload is
// logged but never blocks or stops the proxy.
type SDKConfig struct {
	// Enabled toggles the automatic SDK package upload. Default true.
	Enabled bool `yaml:"enabled"`
}

// PresenceConfig controls the microservice presence handshake with
// HRPAuth (POST /services/presence, the "bonjour" handshake). It
// registers HASkinProxy in HRPAuth's in-process presence registry so
// the main service knows it is online. A failed handshake is logged but
// never blocks or stops the proxy.
type PresenceConfig struct {
	// Enabled toggles the presence handshake. Default true.
	Enabled bool `yaml:"enabled"`
	// Name is the service name registered in HRPAuth. Default "HASkinProxy".
	Name string `yaml:"name"`
	// TTLSeconds is the self-declared lifetime in seconds; <=0 (default)
	// means the record never expires.
	TTLSeconds int `yaml:"ttl_seconds"`
}

var AppConfig Config

// LoadConfig reads configuration from the YAML file at path. Missing
// fields fall back to DefaultConfig() values. If the file does not
// exist (or cannot be read), the defaults are used and a default
// config file is generated in place; a failed write is only logged,
// never fatal. This mirrors WinnerProxy's config.Load behavior.
func LoadConfig(path string) error {
	// Start from defaults so fields missing from config.yaml keep their
	// default values (e.g. presence.enabled defaults to true), instead
	// of silently staying at their zero value.
	AppConfig = DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Config file %s not found, generating a default one...", path)
		if err := SaveConfig(path, AppConfig); err != nil {
			log.Printf("WARN: could not write default config to %s: %v", path, err)
		}
		return nil
	}
	return yaml.Unmarshal(data, &AppConfig)
}

func SaveConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func DefaultConfig() Config {
	c := Config{}
	c.Server.ListenAddr = ":2702"
	c.Server.PublicURL = "http://localhost:2702"
	c.Upstream.BaseURL = "http://localhost:2778"
	c.Upstream.Timeout = 10
	c.Upstream.ClientID = ""
	c.Upstream.ClientSecret = ""
	c.Cache.ProfileTTL = 3600
	c.Cache.TextureTTL = 86400
	c.Cache.MaxSizeMB = 256
	c.Presence = PresenceConfig{
		Enabled: true,
		Name:    "HASkinProxy",
	}
	c.SDK = SDKConfig{
		Enabled: true,
	}
	return c
}
