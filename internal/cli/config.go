package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL string `json:"server_url"`
	Author    string `json:"author"`
}

const defaultServerURL = "http://localhost:8080"

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docshub.json"), nil
}

func defaultConfig() *Config {
	return &Config{ServerURL: defaultServerURL}
}

// LoadConfig reads ~/.docshub.json. If the file does not exist, it returns a
// default config (no error).
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaultConfig(), nil
		}
		return nil, err
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = defaultServerURL
	}
	return cfg, nil
}

// SaveConfig writes cfg to ~/.docshub.json with 0600 permissions.
func SaveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
