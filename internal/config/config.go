package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	repoConfigFileName   = ".gitcherry.yml"
	homeConfigFolderName = "gitcherry"
	homeConfigFileName   = "config.yml"

	defaultOnDuplicate    = "ask"
	defaultPreview        = true
	defaultAutoRefresh    = false
	defaultDefaultBranch  = ""
	defaultMessagePattern = "[Transfer] Moved commits from {source} → {target}\nRange: {range}"

	envOnDuplicate    = "GITCHERRY_ON_DUPLICATE"
	envPreview        = "GITCHERRY_PREVIEW"
	envAutoRefresh    = "GITCHERRY_AUTO_REFRESH"
	envDefaultBranch  = "GITCHERRY_DEFAULT_BRANCH"
	envMessagePattern = "GITCHERRY_MESSAGE_TEMPLATE"
)

// Config captures user-defined behaviour flags for GitCherry.
type Config struct {
	OnDuplicate     string
	Preview         bool
	AutoRefresh     bool
	DefaultBranch   string
	MessageTemplate string
}

// Default returns a configuration populated with built-in defaults.
func Default() *Config {
	return &Config{
		OnDuplicate:     defaultOnDuplicate,
		Preview:         defaultPreview,
		AutoRefresh:     defaultAutoRefresh,
		DefaultBranch:   defaultDefaultBranch,
		MessageTemplate: defaultMessagePattern,
	}
}

// Load resolves configuration based on the provided repository path.
// It checks, in order, for a config file in the repository, within the
// user config directory, then environment variables, and finally falls
// back to defaults.
func Load(path string) (*Config, error) {
	base := Default()

	repoConfigPath, err := resolveRepoConfigPath(path)
	if err != nil {
		return nil, err
	}

	if fileCfg, err := loadFileConfig(repoConfigPath); err != nil {
		return nil, err
	} else if fileCfg != nil {
		fileCfg.applyTo(base)
		return base, nil
	}

	if fileCfg, err := loadHomeConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	} else if fileCfg != nil {
		fileCfg.applyTo(base)
		return base, nil
	}

	if envCfg, err := loadEnvConfig(); err != nil {
		return nil, err
	} else if envCfg != nil {
		envCfg.applyTo(base)
		return base, nil
	}

	return base, nil
}

func resolveRepoConfigPath(path string) (string, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(path, repoConfigFileName), nil
		}
		return path, nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return filepath.Join(path, repoConfigFileName), nil
	}

	return "", err
}

type fileConfig struct {
	OnDuplicate          *string `yaml:"onDuplicate"`
	OnDuplicateSnakeCase *string `yaml:"on_duplicate"`
	Preview              *bool   `yaml:"preview"`
	AutoRefresh          *bool   `yaml:"autoRefresh"`
	AutoRefreshSnakeCase *bool   `yaml:"auto_refresh"`
	DefaultBranch        *string `yaml:"defaultBranch"`
	DefaultBranchSnake   *string `yaml:"default_branch"`
	MessageTemplate      *string `yaml:"messageTemplate"`
	MessageTemplateSnake *string `yaml:"message_template"`
}

func (f *fileConfig) applyTo(cfg *Config) {
	if str := firstString(f.OnDuplicate, f.OnDuplicateSnakeCase); str != nil {
		cfg.OnDuplicate = *str
	}

	if b := firstBool(f.Preview, nil); b != nil {
		cfg.Preview = *b
	}

	if b := firstBool(f.AutoRefresh, f.AutoRefreshSnakeCase); b != nil {
		cfg.AutoRefresh = *b
	}

	if str := firstString(f.DefaultBranch, f.DefaultBranchSnake); str != nil {
		cfg.DefaultBranch = *str
	}

	if str := firstString(f.MessageTemplate, f.MessageTemplateSnake); str != nil {
		cfg.MessageTemplate = *str
	}
}

func firstString(values ...*string) *string {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstBool(values ...*bool) *bool {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func loadFileConfig(path string) (*fileConfig, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadHomeConfig() (*fileConfig, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(dir) == "" {
		return nil, fs.ErrNotExist
	}

	path := filepath.Join(dir, homeConfigFolderName, homeConfigFileName)
	return loadFileConfig(path)
}

func loadEnvConfig() (*fileConfig, error) {
	var cfg fileConfig
	var hasValue bool

	if v, ok := lookupString(envOnDuplicate); ok {
		cfg.OnDuplicate = &v
		hasValue = true
	}

	if b, ok, err := lookupBool(envPreview); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", envPreview, err)
	} else if ok {
		cfg.Preview = &b
		hasValue = true
	}

	if b, ok, err := lookupBool(envAutoRefresh); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", envAutoRefresh, err)
	} else if ok {
		cfg.AutoRefresh = &b
		hasValue = true
	}

	if v, ok := lookupString(envDefaultBranch); ok {
		cfg.DefaultBranch = &v
		hasValue = true
	}

	if v, ok := lookupString(envMessagePattern); ok {
		cfg.MessageTemplate = &v
		hasValue = true
	}

	if !hasValue {
		return nil, nil
	}
	return &cfg, nil
}

func lookupString(key string) (string, bool) {
	if v, ok := os.LookupEnv(key); ok {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	}
	return "", false
}

func lookupBool(key string) (bool, bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false, false, nil
	}

	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return false, false, nil
	}

	b, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, false, err
	}
	return b, true, nil
}
