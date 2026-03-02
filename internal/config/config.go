package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	// Apps is the list of Django apps to compare migrations for.
	Apps []string `yaml:"apps"`

	// MigrateCmd is the command to run for migrations.
	MigrateCmd string `yaml:"migrate_cmd"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Apps:       []string{"organization", "bff_main", "shared"},
		MigrateCmd: "python manage.py migrate",
	}
}

// configFileName is the name of the repository-specific config file.
const configFileName = ".mig-diff.yaml"

// globalConfigDir is the directory name for global config.
const globalConfigDir = "mig-diff"

// globalConfigFile is the name of the global config file.
const globalConfigFile = "config.yaml"

// getGlobalConfigPath returns the path to the global config file.
func getGlobalConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, globalConfigDir, globalConfigFile), nil
}

// Load loads configuration with the following priority (highest to lowest):
// 1. Repository-specific .mig-diff.yaml
// 2. Global ~/.config/mig-diff/config.yaml
// 3. Built-in defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	globalPath, err := getGlobalConfigPath()
	if err == nil {
		if err := mergeConfigFromFile(cfg, globalPath); err != nil {
			return nil, err
		}
	}

	repoConfigPath, err := findRepoConfigFile()
	if err == nil {
		if err := mergeConfigFromFile(cfg, repoConfigPath); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// mergeConfigFromFile reads a config file and merges it into the existing config.
func mergeConfigFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil
	}

	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return err
	}

	if len(fileCfg.Apps) > 0 {
		cfg.Apps = fileCfg.Apps
	}
	if fileCfg.MigrateCmd != "" {
		cfg.MigrateCmd = fileCfg.MigrateCmd
	}

	return nil
}

// findRepoConfigFile searches for the repo-specific config file.
func findRepoConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
