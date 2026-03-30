package main

import (
	"path/filepath"
	"strings"
	"testing"

	appsvc "myclaw/internal/app"
	"myclaw/internal/knowledge"
	"myclaw/internal/projectstate"
	"myclaw/internal/promptlib"
	"myclaw/internal/reminder"
	"myclaw/internal/sessionstate"
	"myclaw/internal/weixin"
)

func TestDesktopSettingsCanBeSavedAndReloaded(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge.json"))
	projectStore := projectstate.NewStore(filepath.Join(root, "project.json"))
	promptStore := promptlib.NewStore(filepath.Join(root, "prompts.json"))
	reminders := reminder.NewManager(reminder.NewStore(filepath.Join(root, "reminders.json")))
	sessionStore := sessionstate.NewStore(filepath.Join(root, "sessions.json"))

	service := appsvc.NewServiceWithRuntime(store, nil, reminders, nil, sessionStore, promptStore)
	bridge := weixin.NewBridge(weixin.NewClient("", ""), service, reminders, weixin.BridgeConfig{DataDir: root})
	app := NewDesktopApp(root, store, promptStore, projectStore, nil, nil, service, sessionStore, reminders, bridge)

	saved, err := app.SaveSettings(AppSettingsInput{
		WeixinHistoryMessages: 22,
		WeixinHistoryRunes:    888,
		WeixinEverythingPath:  `C:\Tools\Everything\es.exe`,
	})
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if saved.WeixinHistoryMessages != 22 || saved.WeixinHistoryRunes != 888 || saved.WeixinEverythingPath != `C:\Tools\Everything\es.exe` {
		t.Fatalf("unexpected saved settings: %#v", saved)
	}

	messages, runes := service.WeixinHistoryLimits()
	if messages != 22 || runes != 888 {
		t.Fatalf("expected live service settings to update, got messages=%d runes=%d", messages, runes)
	}
	if bridge.EverythingPath() != `C:\Tools\Everything\es.exe` {
		t.Fatalf("expected live bridge settings to update, got %q", bridge.EverythingPath())
	}

	reloadedService := appsvc.NewServiceWithRuntime(store, nil, reminders, nil, sessionStore, promptStore)
	reloadedBridge := weixin.NewBridge(weixin.NewClient("", ""), reloadedService, reminders, weixin.BridgeConfig{DataDir: root})
	reloadedApp := NewDesktopApp(root, store, promptStore, projectStore, nil, nil, reloadedService, sessionStore, reminders, reloadedBridge)

	reloaded, err := reloadedApp.GetSettings()
	if err != nil {
		t.Fatalf("get reloaded settings: %v", err)
	}
	if reloaded.WeixinHistoryMessages != 22 || reloaded.WeixinHistoryRunes != 888 || reloaded.WeixinEverythingPath != `C:\Tools\Everything\es.exe` {
		t.Fatalf("unexpected reloaded settings: %#v", reloaded)
	}

	messages, runes = reloadedService.WeixinHistoryLimits()
	if messages != 22 || runes != 888 {
		t.Fatalf("expected persisted service settings to load, got messages=%d runes=%d", messages, runes)
	}
	if reloadedBridge.EverythingPath() != `C:\Tools\Everything\es.exe` {
		t.Fatalf("expected persisted bridge settings to load, got %q", reloadedBridge.EverythingPath())
	}
}

func TestDesktopSettingsRejectNegativeValues(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "knowledge.json"))
	projectStore := projectstate.NewStore(filepath.Join(root, "project.json"))
	promptStore := promptlib.NewStore(filepath.Join(root, "prompts.json"))
	reminders := reminder.NewManager(reminder.NewStore(filepath.Join(root, "reminders.json")))
	sessionStore := sessionstate.NewStore(filepath.Join(root, "sessions.json"))
	service := appsvc.NewServiceWithRuntime(store, nil, reminders, nil, sessionStore, promptStore)
	app := NewDesktopApp(root, store, promptStore, projectStore, nil, nil, service, sessionStore, reminders, nil)

	_, err := app.SaveSettings(AppSettingsInput{
		WeixinHistoryMessages: -1,
		WeixinHistoryRunes:    360,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "不能小于 0") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
