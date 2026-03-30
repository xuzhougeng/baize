package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type desktopSettingsStore struct {
	path string
}

type desktopSettingsFile struct {
	WeixinHistoryMessages int `json:"weixin_history_messages"`
	WeixinHistoryRunes    int `json:"weixin_history_runes"`
}

func newDesktopSettingsStore(dataDir string) *desktopSettingsStore {
	return &desktopSettingsStore{
		path: filepath.Join(dataDir, "settings", "app.json"),
	}
}

func (s *desktopSettingsStore) Load() (desktopSettingsFile, bool, error) {
	if s == nil || s.path == "" {
		return desktopSettingsFile{}, false, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return desktopSettingsFile{}, false, nil
		}
		return desktopSettingsFile{}, false, err
	}

	var cfg desktopSettingsFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return desktopSettingsFile{}, false, err
	}
	if cfg.WeixinHistoryMessages < 0 {
		cfg.WeixinHistoryMessages = 0
	}
	if cfg.WeixinHistoryRunes < 0 {
		cfg.WeixinHistoryRunes = 0
	}
	return cfg, true, nil
}

func (s *desktopSettingsStore) Save(cfg desktopSettingsFile) error {
	if s == nil || s.path == "" {
		return nil
	}
	if cfg.WeixinHistoryMessages < 0 {
		cfg.WeixinHistoryMessages = 0
	}
	if cfg.WeixinHistoryRunes < 0 {
		cfg.WeixinHistoryRunes = 0
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
