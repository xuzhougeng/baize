package modelconfig

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvOverrides(t *testing.T) {
	store := NewStore()

	t.Setenv("MYCLAW_MODEL_PROVIDER", "openai")
	t.Setenv("MYCLAW_MODEL_BASE_URL", "https://example.com/v1/")
	t.Setenv("MYCLAW_MODEL_API_KEY", "env-secret")
	t.Setenv("MYCLAW_MODEL_NAME", "env-model")
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load with env: %v", err)
	}
	if loaded.BaseURL != "https://example.com/v1" {
		t.Fatalf("expected normalized env base url, got %q", loaded.BaseURL)
	}
	if loaded.APIKey != "env-secret" {
		t.Fatalf("expected env api key, got %q", loaded.APIKey)
	}
	if loaded.Model != "env-model" {
		t.Fatalf("expected env model, got %q", loaded.Model)
	}
}

func TestMaskSecret(t *testing.T) {
	t.Parallel()

	if got := MaskSecret(""); got != "(empty)" {
		t.Fatalf("unexpected empty mask: %q", got)
	}
	if got := MaskSecret("12345678"); got != "********" {
		t.Fatalf("unexpected short mask: %q", got)
	}
	if got := MaskSecret("abcdefgh12345678"); got != "abcd********5678" {
		t.Fatalf("unexpected long mask: %q", got)
	}
}

func TestDefaultConfigUsesOpenAIDefaults(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if cfg.Provider != "openai" {
		t.Fatalf("unexpected provider: %q", cfg.Provider)
	}
	if cfg.BaseURL == "" {
		t.Fatal("expected base url")
	}
	store := NewStore()
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if loaded.Provider != "openai" {
		t.Fatalf("unexpected loaded provider: %q", loaded.Provider)
	}
}

func TestLoadReadsSavedConfigFile(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "model", "config.json"))
	if err := store.Save(context.Background(), Config{
		Provider: "openai",
		BaseURL:  "https://example.com/v1/",
		APIKey:   "file-secret",
		Model:    "gpt-file",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.BaseURL != "https://example.com/v1" {
		t.Fatalf("unexpected base url: %q", loaded.BaseURL)
	}
	if loaded.APIKey != "file-secret" || loaded.Model != "gpt-file" {
		t.Fatalf("unexpected loaded config: %#v", loaded)
	}
}

func TestLoadEnvOverridesSavedFile(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "model", "config.json"))
	if err := store.Save(context.Background(), Config{
		Provider: "openai",
		BaseURL:  "https://file.example/v1",
		APIKey:   "file-secret",
		Model:    "file-model",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("MYCLAW_MODEL_BASE_URL", "https://env.example/v1/")
	t.Setenv("MYCLAW_MODEL_API_KEY", "env-secret")

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load with env overrides: %v", err)
	}
	if loaded.BaseURL != "https://env.example/v1" {
		t.Fatalf("expected env base url, got %q", loaded.BaseURL)
	}
	if loaded.APIKey != "env-secret" {
		t.Fatalf("expected env api key, got %q", loaded.APIKey)
	}
	if loaded.Model != "file-model" {
		t.Fatalf("expected file model, got %q", loaded.Model)
	}
}
