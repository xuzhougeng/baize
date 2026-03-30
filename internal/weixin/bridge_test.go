package weixin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestMaybeHandleFileFindSearchAndSelection(t *testing.T) {
	t.Parallel()

	bridge := NewBridge(NewClient("https://unit.test", ""), nil, nil, BridgeConfig{
		DataDir:        t.TempDir(),
		EverythingPath: "es.exe",
	})

	paths := []string{
		`E:\xwechat_files\a.pdf`,
		`E:\xwechat_files\b.pdf`,
	}
	bridge.searchFiles = func(_ context.Context, everythingPath, query string, limit int) ([]string, error) {
		if everythingPath != "es.exe" {
			t.Fatalf("unexpected everything path: %q", everythingPath)
		}
		if query != "单细胞" {
			t.Fatalf("unexpected query: %q", query)
		}
		if limit != findResultLimit {
			t.Fatalf("unexpected limit: %d", limit)
		}
		return paths, nil
	}

	var sentTo, sentToken, sentPath string
	bridge.sendFile = func(_ context.Context, toUserID, contextToken, filePath string) error {
		sentTo = toUserID
		sentToken = contextToken
		sentPath = filePath
		return nil
	}

	msg := WeixinMessage{FromUserID: "user-1", ContextToken: "ctx-1"}
	reply, handled, err := bridge.maybeHandleFileFind(context.Background(), msg, "/find 单细胞")
	if err != nil {
		t.Fatalf("search file: %v", err)
	}
	if !handled {
		t.Fatal("expected /find to be handled")
	}
	if !strings.Contains(reply, "找到 2 个文件") || !strings.Contains(reply, `E:\xwechat_files\b.pdf`) {
		t.Fatalf("unexpected search reply: %q", reply)
	}

	reply, handled, err = bridge.maybeHandleFileFind(context.Background(), msg, "2")
	if err != nil {
		t.Fatalf("select file: %v", err)
	}
	if !handled {
		t.Fatal("expected selection to be handled")
	}
	if sentTo != "user-1" || sentToken != "ctx-1" || sentPath != paths[1] {
		t.Fatalf("unexpected send target: to=%q token=%q path=%q", sentTo, sentToken, sentPath)
	}
	if !strings.Contains(reply, "已通过 ClawBot 发送文件 2") {
		t.Fatalf("unexpected selection reply: %q", reply)
	}
	if _, ok := bridge.pendingFileSelection(weixinSessionID(msg)); ok {
		t.Fatal("expected pending selection to be cleared")
	}
}

func TestInferEverythingQueryFromText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "帮我找单细胞pdf", want: "ext:pdf 单细胞", ok: true},
		{input: "去D盘找今天刚生成的文件", want: "d: dm:today", ok: true},
		{input: "帮我找一下知识库里的结论", want: "", ok: false},
	}

	for _, tc := range cases {
		got, ok := inferEverythingQueryFromText(tc.input)
		if ok != tc.ok {
			t.Fatalf("input=%q expected ok=%v got %v (%q)", tc.input, tc.ok, ok, got)
		}
		if got != tc.want {
			t.Fatalf("input=%q expected query=%q got %q", tc.input, tc.want, got)
		}
	}
}
