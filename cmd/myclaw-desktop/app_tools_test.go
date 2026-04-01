package main

import (
	"path/filepath"
	goruntime "runtime"
	"testing"

	appsvc "myclaw/internal/app"
	"myclaw/internal/knowledge"
	"myclaw/internal/projectstate"
	"myclaw/internal/promptlib"
	"myclaw/internal/reminder"
	"myclaw/internal/sessionstate"
	"myclaw/internal/systemcmd"
)

func TestDesktopListToolsIncludesLocalCapabilities(t *testing.T) {
	t.Parallel()

	app := newDesktopAppForToolsTest(t)

	tools, err := app.ListTools()
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	byName := indexToolsByName(tools)
	if _, ok := byName["local::everything_file_search"]; !ok {
		t.Fatalf("expected everything search tool, got %#v", tools)
	}
	if _, ok := byName["local::list_directory"]; !ok {
		t.Fatalf("expected directory listing tool, got %#v", tools)
	}
	if systemcmd.SupportedForCurrentPlatform() {
		if _, ok := byName["local::readonly_system_command"]; !ok {
			t.Fatalf("expected system inspection tool, got %#v", tools)
		}
	}

	fileSearch := byName["local::everything_file_search"]
	if fileSearch.Title != "文件检索" {
		t.Fatalf("unexpected everything search title: %#v", fileSearch)
	}
	if !fileSearch.Configurable {
		t.Fatalf("expected everything search to be configurable, got %#v", fileSearch)
	}
	if goruntime.GOOS == "windows" {
		if fileSearch.Status != "需配置 es.exe 路径" || fileSearch.StatusTone != "pending" {
			t.Fatalf("unexpected Windows file search status: %#v", fileSearch)
		}
	} else if fileSearch.Status != "当前平台暂不支持" || fileSearch.StatusTone != "off" {
		t.Fatalf("unexpected non-Windows file search status: %#v", fileSearch)
	}
}

func TestDesktopListToolsReflectsConfiguredEverythingPath(t *testing.T) {
	t.Parallel()

	app := newDesktopAppForToolsTest(t)

	_, err := app.SaveSettings(AppSettingsInput{
		WeixinHistoryMessages: 12,
		WeixinHistoryRunes:    360,
		WeixinEverythingPath:  `C:\Tools\Everything\es.exe`,
	})
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}

	tools, err := app.ListTools()
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	fileSearch := indexToolsByName(tools)["local::everything_file_search"]
	if fileSearch.ConfigValue != `C:\Tools\Everything\es.exe` {
		t.Fatalf("unexpected file search config value: %#v", fileSearch)
	}
	if goruntime.GOOS == "windows" {
		if fileSearch.Status != "已就绪" || fileSearch.StatusTone != "on" {
			t.Fatalf("unexpected configured Windows file search status: %#v", fileSearch)
		}
	}
}

func newDesktopAppForToolsTest(t *testing.T) *DesktopApp {
	t.Helper()

	root := t.TempDir()
	store := knowledge.NewStore(filepath.Join(root, "app.db"))
	projectStore := projectstate.NewStore(filepath.Join(root, "app.db"))
	promptStore := promptlib.NewStore(filepath.Join(root, "app.db"))
	reminders := reminder.NewManager(reminder.NewStore(filepath.Join(root, "app.db")))
	sessionStore := sessionstate.NewStore(filepath.Join(root, "app.db"))
	service := appsvc.NewServiceWithRuntime(store, nil, reminders, nil, sessionStore, promptStore)
	return NewDesktopApp(root, store, promptStore, projectStore, nil, nil, service, sessionStore, reminders, nil)
}

func indexToolsByName(items []ToolItem) map[string]ToolItem {
	out := make(map[string]ToolItem, len(items))
	for _, item := range items {
		out[item.Name] = item
	}
	return out
}
