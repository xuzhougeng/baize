package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	appsvc "myclaw/internal/app"
	"myclaw/internal/sqliteutil"
)

const desktopSettingsRowID = "primary"

type desktopSettingsStore struct {
	path     string
	db       *sql.DB
	initOnce sync.Once
	initErr  error
}

type desktopSettingsFile struct {
	WeixinHistoryMessages int               `json:"weixin_history_messages"`
	WeixinHistoryRunes    int               `json:"weixin_history_runes"`
	WeixinEverythingPath  string            `json:"weixin_everything_path"`
	DisabledToolNames     []string          `json:"disabled_tool_names,omitempty"`
	DesktopChatSessions   map[string]string `json:"desktop_chat_sessions,omitempty"`
}

func newDesktopSettingsStore(dataDir string) *desktopSettingsStore {
	return &desktopSettingsStore{
		path: filepath.Join(dataDir, "app.db"),
	}
}

func (s *desktopSettingsStore) Load() (desktopSettingsFile, bool, error) {
	if s == nil || s.path == "" {
		return desktopSettingsFile{}, false, nil
	}
	if err := s.ensureReady(); err != nil {
		return desktopSettingsFile{}, false, err
	}

	row := s.db.QueryRowContext(context.Background(), `
		SELECT weixin_history_messages, weixin_history_runes, weixin_everything_path, disabled_tool_names_json, desktop_chat_sessions_json
		FROM desktop_settings
		WHERE id = ?
	`, desktopSettingsRowID)
	var (
		cfg               desktopSettingsFile
		disabledToolsJSON string
		chatSessionsJSON  string
	)
	if err := row.Scan(&cfg.WeixinHistoryMessages, &cfg.WeixinHistoryRunes, &cfg.WeixinEverythingPath, &disabledToolsJSON, &chatSessionsJSON); err != nil {
		if err == sql.ErrNoRows {
			return desktopSettingsFile{}, false, nil
		}
		return desktopSettingsFile{}, false, err
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(disabledToolsJSON)), &cfg.DisabledToolNames); err != nil {
		return desktopSettingsFile{}, false, err
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(chatSessionsJSON)), &cfg.DesktopChatSessions); err != nil {
		return desktopSettingsFile{}, false, err
	}
	normalizeDesktopSettings(&cfg)
	return cfg, true, nil
}

func (s *desktopSettingsStore) Save(cfg desktopSettingsFile) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := s.ensureReady(); err != nil {
		return err
	}

	normalizeDesktopSettings(&cfg)
	disabledToolsJSON, err := json.Marshal(cfg.DisabledToolNames)
	if err != nil {
		return err
	}
	chatSessionsJSON, err := json.Marshal(cfg.DesktopChatSessions)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `
		INSERT INTO desktop_settings (
			id, weixin_history_messages, weixin_history_runes, weixin_everything_path, disabled_tool_names_json, desktop_chat_sessions_json
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			weixin_history_messages = excluded.weixin_history_messages,
			weixin_history_runes = excluded.weixin_history_runes,
			weixin_everything_path = excluded.weixin_everything_path,
			disabled_tool_names_json = excluded.disabled_tool_names_json,
			desktop_chat_sessions_json = excluded.desktop_chat_sessions_json
	`, desktopSettingsRowID, cfg.WeixinHistoryMessages, cfg.WeixinHistoryRunes, cfg.WeixinEverythingPath, string(disabledToolsJSON), string(chatSessionsJSON))
	return err
}

func (s *desktopSettingsStore) ensureReady() error {
	s.initOnce.Do(func() {
		s.db, s.initErr = sqliteutil.Open(s.path)
		if s.initErr != nil {
			return
		}
		_, s.initErr = s.db.Exec(`
			CREATE TABLE IF NOT EXISTS desktop_settings (
				id TEXT PRIMARY KEY,
				weixin_history_messages INTEGER NOT NULL DEFAULT 0,
				weixin_history_runes INTEGER NOT NULL DEFAULT 0,
				weixin_everything_path TEXT NOT NULL DEFAULT '',
				disabled_tool_names_json TEXT NOT NULL DEFAULT '[]',
				desktop_chat_sessions_json TEXT NOT NULL DEFAULT '{}'
			)
		`)
		if s.initErr != nil {
			return
		}
		if _, err := s.db.Exec(`ALTER TABLE desktop_settings ADD COLUMN disabled_tool_names_json TEXT NOT NULL DEFAULT '[]'`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			s.initErr = err
		}
	})
	return s.initErr
}

func normalizeDesktopSettings(cfg *desktopSettingsFile) {
	if cfg == nil {
		return
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
	cfg.DisabledToolNames = appsvc.NormalizeAgentToolNames(cfg.DisabledToolNames)
	cfg.DesktopChatSessions = normalizeDesktopChatSessions(cfg.DesktopChatSessions)
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
