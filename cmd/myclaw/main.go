package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"myclaw/internal/app"
	"myclaw/internal/knowledge"
	"myclaw/internal/weixin"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	dataDirFlag := flag.String("data-dir", envOrDefault("MYCLAW_DATA_DIR", "data"), "directory used to persist data")
	weixinEnabled := flag.Bool("weixin", envOrDefault("MYCLAW_WEIXIN_ENABLED", "0") == "1", "enable WeChat bridge")
	weixinLogin := flag.Bool("weixin-login", false, "run WeChat QR login and exit")
	weixinLogout := flag.Bool("weixin-logout", false, "remove saved WeChat credentials and exit")
	flag.Parse()

	dataDir, err := filepath.Abs(*dataDirFlag)
	if err != nil {
		log.Fatalf("resolve data dir: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	store := knowledge.NewStore(filepath.Join(dataDir, "knowledge", "entries.json"))
	service := app.NewService(store)
	bridge := weixin.NewBridge(weixin.NewClient("", ""), service, weixin.BridgeConfig{
		DataDir: dataDir,
	})

	if *weixinLogout {
		if err := bridge.Logout(); err != nil {
			log.Fatalf("weixin logout: %v", err)
		}
		log.Printf("weixin credentials removed")
		return
	}

	if *weixinLogin {
		if err := bridge.Login(); err != nil {
			log.Fatalf("weixin login: %v", err)
		}
		log.Printf("weixin login complete")
		return
	}

	if !*weixinEnabled {
		log.Printf("myclaw started without interface; set -weixin or MYCLAW_WEIXIN_ENABLED=1")
		return
	}

	if !bridge.LoadAccount() {
		log.Printf("no saved weixin account found, starting login flow")
		if err := bridge.Login(); err != nil {
			log.Fatalf("weixin login: %v", err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("myclaw started: data_dir=%s interface=weixin", dataDir)
	if err := bridge.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("weixin bridge stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
