package weixin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTextSupportsVoiceFallback(t *testing.T) {
	t.Parallel()

	text := extractText(WeixinMessage{
		ItemList: []MessageItem{
			{Type: ItemTypeVoice, VoiceItem: &VoiceItem{Text: "语音转写内容"}},
		},
	})
	if text != "语音转写内容" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestSplitByRunes(t *testing.T) {
	t.Parallel()

	chunks := splitByRunes("123456789", 4)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0] != "1234" || chunks[1] != "5678" || chunks[2] != "9" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}

func TestSendTextMessageIncludesClientIDAndBaseInfo(t *testing.T) {
	t.Parallel()

	var got SendMessageRequest
	client := newTestClient(t, &got)

	err := client.SendTextMessage(context.Background(), "user-1", "hello", "ctx-1")
	if err != nil {
		t.Fatalf("send text: %v", err)
	}
	if got.Msg.ClientID == "" {
		t.Fatal("expected client id")
	}
	if got.BaseInfo.ChannelVersion != ChannelVersion {
		t.Fatalf("unexpected channel version: %q", got.BaseInfo.ChannelVersion)
	}
	if got.Msg.ContextToken != "ctx-1" {
		t.Fatalf("unexpected context token: %q", got.Msg.ContextToken)
	}
}

func TestFinalizeLoginPersistsAccount(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	bridge := NewBridge(NewClient("https://unit.test", ""), nil, nil, BridgeConfig{DataDir: dataDir})

	account, err := bridge.finalizeLogin(&QRCodeStatusResponse{
		Status:      "confirmed",
		BotToken:    "bot-token",
		BaseURL:     "https://weixin.example",
		ILinkBotID:  "bot-123",
		ILinkUserID: "user-456",
	})
	if err != nil {
		t.Fatalf("finalize login: %v", err)
	}
	if account.AccountID != "bot-123" {
		t.Fatalf("unexpected account id: %q", account.AccountID)
	}

	saved, ok := bridge.ReadSavedAccount()
	if !ok {
		t.Fatal("expected saved account")
	}
	if saved.Token != "bot-token" || saved.BaseURL != "https://weixin.example" {
		t.Fatalf("unexpected saved account: %#v", saved)
	}
}

func TestLoadAccountUsesSavedCredentials(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	accountPath := filepath.Join(dataDir, "weixin-bridge", "account.json")
	if err := os.MkdirAll(filepath.Dir(accountPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(accountPath, []byte(`{"token":"saved-token","base_url":"https://saved.example","account_id":"bot-1"}`), 0o644); err != nil {
		t.Fatalf("write account: %v", err)
	}

	bridge := NewBridge(NewClient("https://unit.test", ""), nil, nil, BridgeConfig{DataDir: dataDir})
	if !bridge.LoadAccount() {
		t.Fatal("expected saved account to load")
	}
	if bridge.client.token != "saved-token" {
		t.Fatalf("unexpected token: %q", bridge.client.token)
	}
	if bridge.client.BaseURL() != "https://saved.example" {
		t.Fatalf("unexpected base URL: %q", bridge.client.BaseURL())
	}
}
