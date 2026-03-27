package modelconfig

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultProvider = "openai"
	DefaultBaseURL  = "https://api.openai.com/v1"
)

type Config struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

type Store struct {
	path string
}

func NewStore(path ...string) *Store {
	store := &Store{}
	if len(path) > 0 {
		store.path = strings.TrimSpace(path[0])
	}
	return store
}

func DefaultConfig() Config {
	return Config{
		Provider: DefaultProvider,
		BaseURL:  DefaultBaseURL,
	}
}

func (s *Store) Load(ctx context.Context) (Config, error) {
	cfg := DefaultConfig()
	saved, ok, err := s.LoadSaved(ctx)
	if err != nil {
		return Config{}, err
	}
	if ok {
		cfg = mergeConfig(cfg, saved)
	}
	applyEnvOverrides(&cfg)
	return cfg.Normalize(), nil
}

func (s *Store) LoadSaved(_ context.Context) (Config, bool, error) {
	if strings.TrimSpace(s.path) == "" {
		return Config{}, false, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, err
	}
	return cfg.Normalize(), true, nil
}

func (s *Store) Save(_ context.Context, cfg Config) error {
	if strings.TrimSpace(s.path) == "" {
		return errors.New("model config store is read-only")
	}

	cfg = cfg.Normalize()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Clear(_ context.Context) error {
	if strings.TrimSpace(s.path) == "" {
		return errors.New("model config store is read-only")
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Store) Path() string {
	return s.path
}

func (c Config) Normalize() Config {
	c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
	if c.Provider == "" {
		c.Provider = DefaultProvider
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.Model = strings.TrimSpace(c.Model)
	return c
}

func (c Config) MissingFields() []string {
	var missing []string
	if strings.TrimSpace(c.Provider) == "" {
		missing = append(missing, "provider")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		missing = append(missing, "base_url")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		missing = append(missing, "api_key")
	}
	if strings.TrimSpace(c.Model) == "" {
		missing = append(missing, "model")
	}
	return missing
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if value := strings.TrimSpace(os.Getenv("MYCLAW_MODEL_PROVIDER")); value != "" {
		cfg.Provider = value
	}
	if value := strings.TrimSpace(os.Getenv("MYCLAW_MODEL_BASE_URL")); value != "" {
		cfg.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("MYCLAW_MODEL_API_KEY")); value != "" {
		cfg.APIKey = value
	}
	if value := strings.TrimSpace(os.Getenv("MYCLAW_MODEL_NAME")); value != "" {
		cfg.Model = value
	}
}

func ActiveEnvOverrides() []string {
	fields := make([]string, 0, 4)
	if strings.TrimSpace(os.Getenv("MYCLAW_MODEL_PROVIDER")) != "" {
		fields = append(fields, "provider")
	}
	if strings.TrimSpace(os.Getenv("MYCLAW_MODEL_BASE_URL")) != "" {
		fields = append(fields, "base_url")
	}
	if strings.TrimSpace(os.Getenv("MYCLAW_MODEL_API_KEY")) != "" {
		fields = append(fields, "api_key")
	}
	if strings.TrimSpace(os.Getenv("MYCLAW_MODEL_NAME")) != "" {
		fields = append(fields, "model")
	}
	return fields
}

func mergeConfig(base Config, override Config) Config {
	if strings.TrimSpace(override.Provider) != "" {
		base.Provider = override.Provider
	}
	if strings.TrimSpace(override.BaseURL) != "" {
		base.BaseURL = override.BaseURL
	}
	if strings.TrimSpace(override.APIKey) != "" {
		base.APIKey = override.APIKey
	}
	if strings.TrimSpace(override.Model) != "" {
		base.Model = override.Model
	}
	return base
}

func MaskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "(empty)"
	}
	runes := []rune(secret)
	if len(runes) <= 8 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:4]) + strings.Repeat("*", len(runes)-8) + string(runes[len(runes)-4:])
}
