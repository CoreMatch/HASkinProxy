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
		// (e.g. http://localhost:2702). It is announced to HRPAuth in the
		// presence handshake (sdk_url) and embedded in the WEBUI SDK so
		// the frontend can locate the CustomSkinLoader setup page.
		// Empty means the SDK falls back to a relative path.
		PublicURL string `yaml:"public_url"`
	} `yaml:"server"`
	Upstream struct {
		BaseURL      string `yaml:"base_url"`
		Timeout      int    `yaml:"timeout"` // in seconds
		ManageToken  string `yaml:"manage_token"`
		EnableManage bool   `yaml:"enable_manage"`
	} `yaml:"upstream"`
	Cache struct {
		ProfileTTL int `yaml:"profile_ttl"` // in seconds
		TextureTTL int `yaml:"texture_ttl"` // in seconds
		MaxSizeMB  int `yaml:"max_size_mb"`
	} `yaml:"cache"`
	Presence PresenceConfig `yaml:"presence"`
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

func LoadConfig(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Config file %s not found, generating a default one...", path)
		AppConfig = DefaultConfig()
		if err := SaveConfig(path, AppConfig); err != nil {
			return err
		}
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
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
	c.Upstream.BaseURL = "http://localhost:2778" // Default upstream URL
	c.Upstream.Timeout = 10
	c.Cache.ProfileTTL = 3600
	c.Cache.TextureTTL = 86400
	c.Cache.MaxSizeMB = 256
	c.Presence = PresenceConfig{
		Enabled: true,
		Name:    "HASkinProxy",
	}
	return c
}
