package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"myclaw/internal/knowledge"
)

func TestHandleMessageRememberAndList(t *testing.T) {
	t.Parallel()

	store := knowledge.NewStore(filepath.Join(t.TempDir(), "entries.json"))
	service := NewService(store)
	ctx := context.Background()

	reply, err := service.HandleMessage(ctx, MessageContext{UserID: "u1", Interface: "weixin"}, "记住：Windows 版本先做微信接口")
	if err != nil {
		t.Fatalf("remember failed: %v", err)
	}
	if !strings.Contains(reply, "已记住") {
		t.Fatalf("unexpected remember reply: %q", reply)
	}

	reply, err = service.HandleMessage(ctx, MessageContext{}, "/list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(reply, "Windows 版本先做微信接口") {
		t.Fatalf("list reply missing entry: %q", reply)
	}
}

func TestHandleMessageQuestionReturnsAllKnowledge(t *testing.T) {
	t.Parallel()

	store := knowledge.NewStore(filepath.Join(t.TempDir(), "entries.json"))
	service := NewService(store)
	ctx := context.Background()

	if _, err := service.HandleMessage(ctx, MessageContext{}, "/remember 未来需要支持 macOS"); err != nil {
		t.Fatalf("remember macos: %v", err)
	}
	if _, err := service.HandleMessage(ctx, MessageContext{}, "/remember 现在只做最小知识库检索"); err != nil {
		t.Fatalf("remember retrieval: %v", err)
	}

	reply, err := service.HandleMessage(ctx, MessageContext{}, "macOS 什么时候做？")
	if err != nil {
		t.Fatalf("question failed: %v", err)
	}
	if !strings.Contains(reply, "我已读取当前知识库的全部内容") {
		t.Fatalf("missing all-knowledge marker: %q", reply)
	}
	if !strings.Contains(reply, "未来需要支持 macOS") {
		t.Fatalf("missing first entry: %q", reply)
	}
	if !strings.Contains(reply, "现在只做最小知识库检索") {
		t.Fatalf("missing second entry: %q", reply)
	}
}
