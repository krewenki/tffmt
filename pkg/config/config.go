// Package config provides configuration functionality for the tffmt application.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Settings holds the configuration options for the formatting tool
type Settings struct {
	Write     *bool `yaml:"write"`
	Check     *bool `yaml:"check"`
	List      *bool `yaml:"list"`
	Diff      *bool `yaml:"diff"`
	Recursive *bool `yaml:"recursive"`
}

// Config holds all configuration and flag values
type Config struct {
	Write     bool
	Check     bool
	List      bool
	Diff      bool
	Recursive bool
	Test      bool
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		Write:     true,
		Check:     false,
		List:      true,
		Diff:      false,
		Recursive: false,
		Test:      false,
	}
}

// FindConfigFile looks for a settings file in the following locations:
// 1. .tffmt.yml in the current directory
// 2. .tffmt.yml in any parent directory
// 3. ~/.config/tffmt/tffmt.yml
// Returns the path to the first file found, or an empty string if none exists.
func FindConfigFile() string {
	// 1. Check current directory
	if _, err := os.Stat(".tffmt.yml"); err == nil {
		return ".tffmt.yml"
	}

	// 2. Check parent directories
	dir, err := os.Getwd()
	if err == nil {
		for {
			parentDir := filepath.Dir(dir)
			if parentDir == dir {
				// We've reached the root directory
				break
			}

			path := filepath.Join(parentDir, ".tffmt.yml")
			if _, err := os.Stat(path); err == nil {
				return path
			}

			dir = parentDir
		}
	}

	// 3. Check user's home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(homeDir, ".config", "tffmt", "tffmt.yml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// No settings file found
	return ""
}

// LoadSettings attempts to load settings from a config file
func LoadSettings() (Settings, error) {
	settings := Settings{}

	configPath := FindConfigFile()
	if configPath == "" {
		// No config file found, return defaults
		return settings, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return settings, err
	}

	err = yaml.Unmarshal(data, &settings)
	if err != nil {
		return settings, fmt.Errorf("error parsing %s: %w", configPath, err)
	}

	return settings, nil
}

// ApplySettings updates the Config with values from the settings file
// but only for flags that were not explicitly set on the command line
func ApplySettings(c *Config, s Settings, passedFlags map[string]bool) {
	// Only apply settings if they were specified in the config
	// and the corresponding flag was not explicitly set on the command line
	if s.Write != nil && !passedFlags["write"] {
		c.Write = *s.Write
	}
	if s.Check != nil && !passedFlags["check"] {
		c.Check = *s.Check
	}
	if s.List != nil && !passedFlags["list"] {
		c.List = *s.List
	}
	if s.Diff != nil && !passedFlags["diff"] {
		c.Diff = *s.Diff
	}
	if s.Recursive != nil && !passedFlags["recursive"] {
		c.Recursive = *s.Recursive
	}
}
