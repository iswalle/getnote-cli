package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultClientID  = "getnote-cli"
	DefaultAPIBaseURL = "https://openapi.biji.com"
)

// Config holds the CLI configuration.
type Config struct {
	APIKey   string `json:"api_key"`
	ClientID string `json:"client_id"`
}

var (
	instance *Config
	once     sync.Once
)

// Get returns the singleton config, loading from file on first call.
func Get() *Config {
	once.Do(func() {
		instance = &Config{
			ClientID: DefaultClientID,
		}
		_ = instance.load()
	})
	return instance
}

// configPath returns the path to the config file (~/.getnote/config.json).
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".getnote", "config.json"), nil
}

// load reads the config from disk. Missing file is not an error.
func (c *Config) load() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

// Save writes the current config to disk, creating the directory if needed.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Clear removes the stored API key and saves the config.
func (c *Config) Clear() error {
	c.APIKey = ""
	return c.Save()
}

// IsLoggedIn reports whether an API key is configured.
func (c *Config) IsLoggedIn() bool {
	return c.APIKey != ""
}
