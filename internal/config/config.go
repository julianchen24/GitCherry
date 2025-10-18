package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var (
	// ErrConfigNotFound is returned when no configuration file can be located.
	ErrConfigNotFound = errors.New("config not found")
)

const (
	defaultFileName = "config.yaml"
	envConfigPath   = "GITCHERRY_CONFIG"
)

// Config represents the user-provided settings that shape GitCherry's behaviour.
type Config struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

// Profile groups cherry-pick preferences for a repository.
type Profile struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Branches    []string `yaml:"branches"`
}

// Default provides a usable in-memory configuration when no file exists yet.
func Default() *Config {
	return &Config{
		Profiles: map[string]Profile{
			"default": {
				Name:        "default",
				Description: "Default GitCherry profile",
				Branches:    []string{"main"},
			},
		},
	}
}

// LoadDefault discovers the configuration file path and loads it.
func LoadDefault() (*Config, error) {
	if override := os.Getenv(envConfigPath); override != "" {
		return Load(override)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(configDir, "gitcherry", defaultFileName)
	return Load(path)
}

// Load reads and decodes configuration data stored in YAML format.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
