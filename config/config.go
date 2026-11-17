package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adrg/xdg"
)

const configPath = "cloudquery/config.json"

var configKeys = []string{
	"team",
	"team_internal",
	"first_sync_completed",
}

// SetConfigHome sets the configuration home directory - useful for testing
func SetConfigHome(configDir string) error {
	return setXDGEnv("XDG_CONFIG_HOME", configDir)
}

// UnsetConfigHome unsets the configuration home directory returning to the default value
func UnsetConfigHome() error {
	return unsetXDGEnv("XDG_CONFIG_HOME")
}

// GetValue reads the value of a config key from the config file
func GetValue(key string) (string, error) {
	if !slices.Contains(configKeys, key) {
		return "", fmt.Errorf("invalid config key %v (options are: %v)", key, strings.Join(configKeys, ", "))
	}
	configFilePath, err := xdg.ConfigFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to get config file path: %w", err)
	}
	b, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}
	var config map[string]any
	err = json.Unmarshal(b, &config)
	if err != nil {
		return "", fmt.Errorf("failed to parse config file: %w", err)
	}
	if _, ok := config[key]; !ok {
		return "", nil
	}
	return config[key].(string), nil
}

// SetValue updates the value of a config key in the config file
func SetValue(key, val string) error {
	return setValue(key, &val)
}

// UnsetValue removes the value of a config key from the config file
func UnsetValue(key string) error {
	return setValue(key, nil)
}

func setValue(key string, val *string) error {
	if !slices.Contains(configKeys, key) {
		return fmt.Errorf("invalid config key %v (options are: %v)", key, strings.Join(configKeys, ", "))
	}
	configFilePath, err := xdg.ConfigFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to get config file path: %w", err)
	}
	var config map[string]any
	b, err := os.ReadFile(configFilePath)
	switch {
	case err == nil:
		err = json.Unmarshal(b, &config)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	case os.IsNotExist(err):
		config = make(map[string]any)
	default:
		return fmt.Errorf("failed to read config file: %w", err)
	}
	if val == nil {
		// unset
		delete(config, key)
	} else {
		// set
		config[key] = val
	}
	b, err = json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	err = writeFileAtomically(configFilePath, b)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// SetDataHome sets the data home directory - useful for testing
func SetDataHome(dataHome string) error {
	return setXDGEnv("XDG_DATA_HOME", dataHome)
}

// UnsetDataHome unsets the data home directory returning to the default value
func UnsetDataHome() error {
	return unsetXDGEnv("XDG_DATA_HOME")
}

// SaveDataString saves a string to a file in the data home directory
func SaveDataString(relPath string, data string) error {
	filePath, err := xdg.DataFile(relPath)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}
	if err := writeFileAtomically(filePath, []byte(data)); err != nil {
		return fmt.Errorf("failed to write data to %q: %w", filePath, err)
	}
	return nil
}

func writeFileAtomically(filePath string, data []byte) error {
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), "."+filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	if err := writeSyncAndClose(tempFile, data); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func writeSyncAndClose(file *os.File, data []byte) error {
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

// ReadDataString reads a string from a file in the data home directory
func ReadDataString(relPath string) (string, error) {
	filePath, err := xdg.DataFile(relPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file path: %w", err)
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// DeleteDataString deletes a file in the data home directory
func DeleteDataString(relPath string) error {
	filePath, err := xdg.DataFile(relPath)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}
	if err := os.RemoveAll(filePath); err != nil {
		return fmt.Errorf("failed to remove file %q: %w", filePath, err)
	}
	return nil
}

func setXDGEnv(key, value string) error {
	err := os.Setenv(key, value)
	if err != nil {
		return fmt.Errorf("failed to set %s: %w", key, err)
	}
	xdg.Reload()
	return nil
}

func unsetXDGEnv(key string) error {
	err := os.Unsetenv(key)
	if err != nil {
		return fmt.Errorf("failed to unset %s: %w", key, err)
	}
	xdg.Reload()
	return nil
}
