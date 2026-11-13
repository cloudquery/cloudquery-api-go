package config

import (
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/adrg/xdg"
)

func TestSetGetValue(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()

	err := SetValue("team", "my-team")
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetValue("team")
	if err != nil {
		t.Fatal(err)
	}
	if got != "my-team" {
		t.Fatalf("expected %q, got %q", "my-team", got)
	}

	err = UnsetValue("team")
	if err != nil {
		t.Fatal(err)
	}

	got, err = GetValue("team")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected %q, got %q", "", got)
	}
}

func TestSetConfigHome(t *testing.T) {
	r := require.New(t)
	configDir := t.TempDir()

	err := SetConfigHome(configDir)
	r.NoError(err)

	r.Equal(configDir, xdg.ConfigHome)

	err = SetValue("team", "set-config-test-value")
	r.NoError(err)

	// check that the config file was created in the temporary directory,
	// not somewhere else
	_, err = os.Stat(path.Join(configDir, "cloudquery", "config.json"))
	r.NoError(err)

	err = UnsetConfigHome()
	r.NoError(err)

	// check that we are no longer set to the temporary directory
	r.NotEqual(configDir, xdg.ConfigHome)
}

func TestSetDataHome(t *testing.T) {
	r := require.New(t)
	configDir := t.TempDir()

	err := SetDataHome(configDir)
	r.NoError(err)

	r.Equal(configDir, xdg.DataHome)

	err = SaveDataString("cloudquery/token", "my-token")
	r.NoError(err)

	// check that the config file was created in the temporary directory,
	// not somewhere else
	_, err = os.Stat(path.Join(configDir, "cloudquery", "token"))
	r.NoError(err)

	err = UnsetDataHome()
	r.NoError(err)

	// check that we are no longer set to the temporary directory
	r.NotEqual(configDir, xdg.DataHome)
}

func skipFileModeTestOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix file permission bits")
	}
}

func useTempDataHome(t *testing.T) string {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	xdg.Reload()
	return dataHome
}

func requireOwnerOnlyMode(t *testing.T, filePath string) {
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSaveDataStringCreatesOwnerOnlyFile(t *testing.T) {
	skipFileModeTestOnWindows(t)
	dataHome := useTempDataHome(t)

	require.NoError(t, SaveDataString("cloudquery/token", "my-token"))

	requireOwnerOnlyMode(t, filepath.Join(dataHome, "cloudquery", "token"))
	got, err := ReadDataString("cloudquery/token")
	require.NoError(t, err)
	require.Equal(t, "my-token", got)
}

func TestSaveDataStringReplacesWorldReadableFileWithOwnerOnlyFile(t *testing.T) {
	skipFileModeTestOnWindows(t)
	dataHome := useTempDataHome(t)
	tokenPath := filepath.Join(dataHome, "cloudquery", "token")
	require.NoError(t, os.MkdirAll(filepath.Dir(tokenPath), 0o700))
	require.NoError(t, os.WriteFile(tokenPath, []byte("old-token"), 0o644))
	require.NoError(t, os.Chmod(tokenPath, 0o644))

	require.NoError(t, SaveDataString("cloudquery/token", "new-token"))

	requireOwnerOnlyMode(t, tokenPath)
	got, err := ReadDataString("cloudquery/token")
	require.NoError(t, err)
	require.Equal(t, "new-token", got)
}

func TestSaveDataStringKeepsExistingFileWhenWriteFails(t *testing.T) {
	skipFileModeTestOnWindows(t)
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so the write cannot be made to fail")
	}
	dataHome := useTempDataHome(t)
	tokenDir := filepath.Join(dataHome, "cloudquery")
	tokenPath := filepath.Join(tokenDir, "token")
	require.NoError(t, os.MkdirAll(tokenDir, 0o700))
	require.NoError(t, os.WriteFile(tokenPath, []byte("old-token"), 0o600))
	require.NoError(t, os.Chmod(tokenDir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(tokenDir, 0o700) })

	require.Error(t, SaveDataString("cloudquery/token", "new-token"))

	got, err := os.ReadFile(tokenPath)
	require.NoError(t, err)
	require.Equal(t, "old-token", string(got))
	entries, err := os.ReadDir(tokenDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestSetValueCreatesOwnerOnlyFile(t *testing.T) {
	skipFileModeTestOnWindows(t)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	xdg.Reload()

	require.NoError(t, SetValue("team", "my-team"))

	requireOwnerOnlyMode(t, filepath.Join(configHome, "cloudquery", "config.json"))
}
