// Package config handles TOML configuration, environment variable overrides,
// and CLI flag integration for llm-cli.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all configuration values.
type Config struct {
	API   APIConfig   `toml:"api"`
	Model ModelConfig `toml:"model"`
}

// APIConfig holds API connection settings.
type APIConfig struct {
	BaseURL                string `toml:"base_url"`
	APIKey                 string `toml:"api_key"`
	TimeoutSeconds         int    `toml:"timeout_seconds"`
	ResponseFormatStrategy string `toml:"response_format_strategy"`
}

// ModelConfig holds model settings.
type ModelConfig struct {
	Name string `toml:"name"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		API: APIConfig{
			BaseURL:                "http://localhost:1234/v1",
			TimeoutSeconds:         120,
			ResponseFormatStrategy: "auto",
		},
	}
}

// Load reads the config file, applies environment variable overrides, and
// returns the merged configuration.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	cfgPath, err := resolve(path)
	if err != nil {
		applyEnv(&cfg)
		return cfg, nil // no config file — use defaults + env
	}

	if err := checkPermissions(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}

	applyEnv(&cfg)
	return cfg, nil
}

// resolve determines the config file path. If path is empty, it tries the
// default location (~/.config/llm-cli/config.toml).
func resolve(path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config file not found: %w", err)
		}
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	defaultPath := filepath.Join(home, ".config", "llm-cli", "config.toml")
	if _, err := os.Stat(defaultPath); err != nil {
		return "", err
	}
	return defaultPath, nil
}

// checkPermissions warns if the config file has overly permissive access.
func checkPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		return fmt.Errorf("config file %s has permissions %04o; expected 0600.\n  The file may contain credentials. Run: chmod 600 %s", path, perm, path)
	}
	return nil
}

// applyEnv overrides config values with environment variables.
func applyEnv(cfg *Config) {
	if v := os.Getenv("LLM_CLI_BASE_URL"); v != "" {
		cfg.API.BaseURL = v
	}
	if v := os.Getenv("LLM_CLI_API_KEY"); v != "" {
		cfg.API.APIKey = v
	}
	if v := os.Getenv("LLM_CLI_MODEL"); v != "" {
		cfg.Model.Name = v
	}
	if v := os.Getenv("LLM_CLI_RESPONSE_FORMAT_STRATEGY"); v != "" {
		cfg.API.ResponseFormatStrategy = v
	}
}
