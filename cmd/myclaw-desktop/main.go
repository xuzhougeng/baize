package main

import (
	"embed"
	"flag"
	"log"
	"path/filepath"

	"myclaw/internal/ai"
	appsvc "myclaw/internal/app"
	"myclaw/internal/knowledge"
	"myclaw/internal/modelconfig"
	"myclaw/internal/reminder"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	dataDirFlag := flag.String("data-dir", envOrDefault("MYCLAW_DATA_DIR", "data"), "directory used to persist data")
	logFileFlag := flag.String("log-file", envOrDefault("MYCLAW_LOG_FILE", ""), "optional log file path")
	flag.Parse()

	if err := configureLogging(*logFileFlag); err != nil {
		log.Fatalf("configure logging: %v", err)
	}

	dataDir, err := prepareDataDir(*dataDirFlag)
	if err != nil {
		log.Fatalf("prepare data dir: %v", err)
	}

	store := knowledge.NewStore(filepath.Join(dataDir, "knowledge", "entries.json"))
	modelStore := modelconfig.NewStore()
	aiService := ai.NewService(modelStore)
	reminderStore := reminder.NewStore(filepath.Join(dataDir, "reminders", "items.json"))
	reminderManager := reminder.NewManager(reminderStore)
	service := appsvc.NewService(store, aiService, reminderManager)
	desktopApp := NewDesktopApp(dataDir, store, aiService, service, reminderManager)

	err = wails.Run(&options.App{
		Title:     "myclaw",
		Width:     1440,
		Height:    960,
		MinWidth:  1120,
		MinHeight: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:         options.NewRGB(244, 238, 228),
		EnableDefaultContextMenu: true,
		OnStartup:                desktopApp.startup,
		OnShutdown:               desktopApp.shutdown,
		Bind: []interface{}{
			desktopApp,
		},
	})
	if err != nil {
		log.Fatalf("run desktop app: %v", err)
	}
}
