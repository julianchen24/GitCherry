package config

import (
	"path/filepath"
	"runtime"
	"testing"

	"os"

	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	resetUserEnv(t, dir)
	clearConfigEnv(t)

	cfg, err := Load(dir)
	require.NoError(t, err)

	require.Equal(t, Default(), cfg)
}

func TestLoadPartialOverrides(t *testing.T) {
	dir := t.TempDir()
	resetUserEnv(t, dir)
	clearConfigEnv(t)

	content := "preview: false\nmessageTemplate: \"Custom {source}\"\n"
	err := os.WriteFile(filepath.Join(dir, ".gitcherry.yml"), []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(dir)
	require.NoError(t, err)

	require.Equal(t, "ask", cfg.OnDuplicate)
	require.False(t, cfg.Preview)
	require.False(t, cfg.AutoRefresh)
	require.Equal(t, "", cfg.DefaultBranch)
	require.Equal(t, "Custom {source}", cfg.MessageTemplate)
}

func TestLoadFullConfig(t *testing.T) {
	dir := t.TempDir()
	resetUserEnv(t, dir)
	clearConfigEnv(t)

	content := `
onDuplicate: replace
preview: false
autoRefresh: true
defaultBranch: main
messageTemplate: |
  [Custom] {source} -> {target}
`
	err := os.WriteFile(filepath.Join(dir, ".gitcherry.yml"), []byte(content), 0o600)
	require.NoError(t, err)

	cfg, err := Load(dir)
	require.NoError(t, err)

	require.Equal(t, "replace", cfg.OnDuplicate)
	require.False(t, cfg.Preview)
	require.True(t, cfg.AutoRefresh)
	require.Equal(t, "main", cfg.DefaultBranch)
	require.Equal(t, "[Custom] {source} -> {target}\n", cfg.MessageTemplate)
}

func TestLoadFallsBackToEnvWhenNoFiles(t *testing.T) {
	dir := t.TempDir()
	resetUserEnv(t, dir)
	clearConfigEnv(t)

	t.Setenv("GITCHERRY_ON_DUPLICATE", "skip")
	t.Setenv("GITCHERRY_PREVIEW", "false")
	t.Setenv("GITCHERRY_AUTO_REFRESH", "true")
	t.Setenv("GITCHERRY_DEFAULT_BRANCH", "develop")
	t.Setenv("GITCHERRY_MESSAGE_TEMPLATE", "{source}->{target}")

	cfg, err := Load(dir)
	require.NoError(t, err)

	require.Equal(t, "skip", cfg.OnDuplicate)
	require.False(t, cfg.Preview)
	require.True(t, cfg.AutoRefresh)
	require.Equal(t, "develop", cfg.DefaultBranch)
	require.Equal(t, "{source}->{target}", cfg.MessageTemplate)
}

func resetUserEnv(t *testing.T, dir string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
		t.Setenv("LOCALAPPDATA", filepath.Join(dir, "localappdata"))
	}

	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GITCHERRY_ON_DUPLICATE", "")
	t.Setenv("GITCHERRY_PREVIEW", "")
	t.Setenv("GITCHERRY_AUTO_REFRESH", "")
	t.Setenv("GITCHERRY_DEFAULT_BRANCH", "")
	t.Setenv("GITCHERRY_MESSAGE_TEMPLATE", "")
}
