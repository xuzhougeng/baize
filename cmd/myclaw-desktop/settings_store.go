package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type desktopSettingsStore struct {
	path string
}

type desktopSettingsFile struct {
	WeixinHistoryMessages int               `json:"weixin_history_messages"`
	WeixinHistoryRunes    int               `json:"weixin_history_runes"`
	WeixinEverythingPath  string            `json:"weixin_everything_path"`
	DesktopChatSessions   map[string]string `json:"desktop_chat_sessions,omitempty"`
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
	cfg.WeixinEverythingPath = filepath.Clean(strings.TrimSpace(cfg.WeixinEverythingPath))
	if cfg.WeixinEverythingPath == "." {
		cfg.WeixinEverythingPath = ""
	}
	cfg.DesktopChatSessions = normalizeDesktopChatSessions(cfg.DesktopChatSessions)
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
	cfg.WeixinEverythingPath = filepath.Clean(strings.TrimSpace(cfg.WeixinEverythingPath))
	if cfg.WeixinEverythingPath == "." {
		cfg.WeixinEverythingPath = ""
	}
	cfg.DesktopChatSessions = normalizeDesktopChatSessions(cfg.DesktopChatSessions)

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func normalizeDesktopChatSessions(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for project, sessionID := range raw {
		project = strings.ToLower(strings.TrimSpace(project))
		sessionID = strings.TrimSpace(sessionID)
		if project == "" || sessionID == "" {
			continue
		}
		out[project] = sessionID
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
