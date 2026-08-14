package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		ListenAddr string `yaml:"listen_addr"`
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
	c.Upstream.BaseURL = "http://localhost:2778" // Default upstream URL
	c.Upstream.Timeout = 10
	c.Cache.ProfileTTL = 3600
	c.Cache.TextureTTL = 86400
	c.Cache.MaxSizeMB = 256
	return c
}
