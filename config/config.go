package config

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/adrg/xdg"
)

const configPath = "cloudquery/config.json"

var configKeys = []string{
	"team",
}

// SetConfigHome sets the configuration home directory - useful for testing
func SetConfigHome(configDir string) error {
	if err := os.Setenv("XDG_CONFIG_HOME", configDir); err != nil {
		return fmt.Errorf("failed to set XDG_CONFIG_HOME: %w", err)
	}
	xdg.Reload()
	return nil
}

// UnsetConfigHome unsets the configuration home directory returning to the default value
func UnsetConfigHome() error {
	if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
		return fmt.Errorf("failed to unset XDG_CONFIG_HOME: %w", err)
	}
	xdg.Reload()
	return nil
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
	err = os.WriteFile(configFilePath, b, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}