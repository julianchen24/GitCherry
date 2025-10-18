package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadUsesOverridePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := []byte(`profiles:
  example:
    name: example
    description: Sample profile
    branches: ["main"]
`)
	err := os.WriteFile(path, content, 0o600)
	require.NoError(t, err)

	t.Setenv("GITCHERRY_CONFIG", path)

	cfg, err := LoadDefault()
	require.NoError(t, err)
	require.Contains(t, cfg.Profiles, "example")
	require.Equal(t, "Sample profile", cfg.Profiles["example"].Description)
}
