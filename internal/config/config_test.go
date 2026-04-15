package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.API.BaseURL != "http://localhost:1234/v1" {
		t.Errorf("BaseURL = %q, want %q", cfg.API.BaseURL, "http://localhost:1234/v1")
	}
	if cfg.API.TimeoutSeconds != 120 {
		t.Errorf("TimeoutSeconds = %d, want 120", cfg.API.TimeoutSeconds)
	}
	if cfg.API.ResponseFormatStrategy != "auto" {
		t.Errorf("ResponseFormatStrategy = %q, want %q", cfg.API.ResponseFormatStrategy, "auto")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `[api]
base_url = "http://localhost:11434/v1"
api_key = "test-key"
timeout_seconds = 60
response_format_strategy = "prompt"

[model]
name = "test-model"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.API.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("BaseURL = %q, want Ollama URL", cfg.API.BaseURL)
	}
	if cfg.API.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.API.APIKey, "test-key")
	}
	if cfg.API.TimeoutSeconds != 60 {
		t.Errorf("TimeoutSeconds = %d, want 60", cfg.API.TimeoutSeconds)
	}
	if cfg.API.ResponseFormatStrategy != "prompt" {
		t.Errorf("ResponseFormatStrategy = %q, want %q", cfg.API.ResponseFormatStrategy, "prompt")
	}
	if cfg.Model.Name != "test-model" {
		t.Errorf("Model.Name = %q, want %q", cfg.Model.Name, "test-model")
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	// Should return defaults when file not found.
	if cfg.API.BaseURL != "http://localhost:1234/v1" {
		t.Errorf("BaseURL = %q, want default", cfg.API.BaseURL)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("LLM_CLI_BASE_URL", "http://env-host:8080/v1")
	t.Setenv("LLM_CLI_API_KEY", "env-key")
	t.Setenv("LLM_CLI_MODEL", "env-model")
	t.Setenv("LLM_CLI_RESPONSE_FORMAT_STRATEGY", "native")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.API.BaseURL != "http://env-host:8080/v1" {
		t.Errorf("BaseURL = %q, want env value", cfg.API.BaseURL)
	}
	if cfg.API.APIKey != "env-key" {
		t.Errorf("APIKey = %q, want env value", cfg.API.APIKey)
	}
	if cfg.Model.Name != "env-model" {
		t.Errorf("Model.Name = %q, want env value", cfg.Model.Name)
	}
	if cfg.API.ResponseFormatStrategy != "native" {
		t.Errorf("ResponseFormatStrategy = %q, want env value", cfg.API.ResponseFormatStrategy)
	}
}

func TestCheckPermissions(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(cfgPath, []byte("[api]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := checkPermissions(cfgPath)
	if err == nil {
		t.Error("expected permission warning for 0644, got nil")
	}
}
