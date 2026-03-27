package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"myclaw/internal/ai"
	appsvc "myclaw/internal/app"
	"myclaw/internal/fileingest"
	"myclaw/internal/knowledge"
	"myclaw/internal/reminder"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	desktopInterface         = "desktop"
	desktopUserID            = "primary"
	maxKnowledgePreviewRunes = 180
)

type DesktopApp struct {
	ctx            context.Context
	dataDir        string
	store          *knowledge.Store
	aiService      *ai.Service
	service        *appsvc.Service
	reminders      *reminder.Manager
	reminderCancel context.CancelFunc
	dialogMu       sync.Mutex
}

type Overview struct {
	DataDir        string `json:"dataDir"`
	AIAvailable    bool   `json:"aiAvailable"`
	AIMessage      string `json:"aiMessage"`
	KnowledgeCount int    `json:"knowledgeCount"`
}

type KnowledgeItem struct {
	ID             string   `json:"id"`
	ShortID        string   `json:"shortId"`
	Text           string   `json:"text"`
	Preview        string   `json:"preview"`
	Source         string   `json:"source"`
	RecordedAt     string   `json:"recordedAt"`
	RecordedAtUnix int64    `json:"recordedAtUnix"`
	Keywords       []string `json:"keywords"`
	IsFile         bool     `json:"isFile"`
}

type KnowledgeMutation struct {
	Message string        `json:"message"`
	Item    KnowledgeItem `json:"item"`
}

type MessageResult struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply     string `json:"reply"`
	Timestamp string `json:"timestamp"`
}

type reminderNotifier struct {
	app *DesktopApp
}

func NewDesktopApp(dataDir string, store *knowledge.Store, aiService *ai.Service, service *appsvc.Service, reminders *reminder.Manager) *DesktopApp {
	return &DesktopApp{
		dataDir:   dataDir,
		store:     store,
		aiService: aiService,
		service:   service,
		reminders: reminders,
	}
}

func (a *DesktopApp) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.WindowCenter(ctx)

	if a.reminders == nil {
		return
	}

	target := reminder.Target{
		Interface: desktopInterface,
		UserID:    desktopUserID,
	}
	a.reminders.RegisterNotifier(target, reminderNotifier{app: a})

	runCtx, cancel := context.WithCancel(context.Background())
	a.reminderCancel = cancel
	go func() {
		if err := a.reminders.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("reminder scheduler stopped: %v", err)
			if a.ctx != nil {
				runtime.LogErrorf(a.ctx, "reminder scheduler stopped: %v", err)
			}
		}
	}()
}

func (a *DesktopApp) shutdown(context.Context) {
	if a.reminderCancel != nil {
		a.reminderCancel()
	}
}

func (n reminderNotifier) Notify(_ context.Context, reminderItem reminder.Reminder) error {
	return n.app.showReminderDialog(reminderItem)
}

func (a *DesktopApp) GetOverview() (Overview, error) {
	entries, err := a.store.List(context.Background())
	if err != nil {
		return Overview{}, err
	}
	available, message, err := a.aiStatus(context.Background())
	if err != nil {
		return Overview{}, err
	}
	return Overview{
		DataDir:        a.dataDir,
		AIAvailable:    available,
		AIMessage:      message,
		KnowledgeCount: len(entries),
	}, nil
}

func (a *DesktopApp) ListKnowledge() ([]KnowledgeItem, error) {
	entries, err := a.store.List(context.Background())
	if err != nil {
		return nil, err
	}
	reverseKnowledge(entries)

	result := make([]KnowledgeItem, 0, len(entries))
	for _, entry := range entries {
		result = append(result, toKnowledgeItem(entry))
	}
	return result, nil
}

func (a *DesktopApp) CreateKnowledge(text string) (KnowledgeMutation, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return KnowledgeMutation{}, errors.New("请输入要保存的记忆内容。")
	}

	entry, err := a.store.Add(context.Background(), knowledge.Entry{
		Text:       text,
		Source:     desktopSourceLabel(),
		RecordedAt: time.Now(),
	})
	if err != nil {
		return KnowledgeMutation{}, err
	}
	return KnowledgeMutation{
		Message: fmt.Sprintf("已记住 #%s", shortID(entry.ID)),
		Item:    toKnowledgeItem(entry),
	}, nil
}

func (a *DesktopApp) AppendKnowledge(idOrPrefix, addition string) (KnowledgeMutation, error) {
	idOrPrefix = strings.TrimSpace(idOrPrefix)
	addition = strings.TrimSpace(addition)
	if idOrPrefix == "" {
		return KnowledgeMutation{}, errors.New("请选择要补充的记忆。")
	}
	if addition == "" {
		return KnowledgeMutation{}, errors.New("请输入补充内容。")
	}

	entry, ok, err := a.store.Append(context.Background(), idOrPrefix, addition)
	if err != nil {
		return KnowledgeMutation{}, err
	}
	if !ok {
		return KnowledgeMutation{}, fmt.Errorf("没有找到知识 #%s。", strings.TrimPrefix(idOrPrefix, "#"))
	}
	return KnowledgeMutation{
		Message: fmt.Sprintf("已补充 #%s", shortID(entry.ID)),
		Item:    toKnowledgeItem(entry),
	}, nil
}

func (a *DesktopApp) DeleteKnowledge(idOrPrefix string) (MessageResult, error) {
	entry, ok, err := a.store.Remove(context.Background(), idOrPrefix)
	if err != nil {
		return MessageResult{}, err
	}
	if !ok {
		return MessageResult{}, fmt.Errorf("没有找到知识 #%s。", strings.TrimPrefix(strings.TrimSpace(idOrPrefix), "#"))
	}
	return MessageResult{
		Message: fmt.Sprintf("已删除 #%s", shortID(entry.ID)),
	}, nil
}

func (a *DesktopApp) ClearKnowledge() (MessageResult, error) {
	if err := a.store.Clear(context.Background()); err != nil {
		return MessageResult{}, err
	}
	return MessageResult{Message: "知识库已清空。"}, nil
}

func (a *DesktopApp) ConfirmAction(title, message string) (bool, error) {
	if a.ctx == nil {
		return false, errors.New("桌面上下文尚未初始化")
	}

	result, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         strings.TrimSpace(title),
		Message:       strings.TrimSpace(message),
		DefaultButton: "No",
	})
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(result)) {
	case "yes", "ok":
		return true, nil
	default:
		return false, nil
	}
}

func (a *DesktopApp) OpenImportDialog() (string, error) {
	if a.ctx == nil {
		return "", errors.New("桌面上下文尚未初始化")
	}

	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择要导入的图片或 PDF",
		DefaultDirectory: a.defaultDialogDirectory(),
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Image / PDF Files",
				Pattern:     "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.pdf",
			},
		},
	})
}

func (a *DesktopApp) ImportFile(path string) (KnowledgeMutation, error) {
	entry, err := a.ingestFile(context.Background(), path)
	if err != nil {
		return KnowledgeMutation{}, err
	}
	return KnowledgeMutation{
		Message: fmt.Sprintf("已导入文件并写入 #%s", shortID(entry.ID)),
		Item:    toKnowledgeItem(entry),
	}, nil
}

func (a *DesktopApp) SendMessage(input string) (ChatResponse, error) {
	reply, err := a.service.HandleMessage(context.Background(), desktopMessageContext(), input)
	if err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{
		Reply:     reply,
		Timestamp: time.Now().Local().Format("2006-01-02 15:04:05"),
	}, nil
}

func (a *DesktopApp) aiStatus(ctx context.Context) (bool, string, error) {
	if a.aiService == nil {
		return false, "模型尚未启用。", nil
	}

	configured, err := a.aiService.IsConfigured(ctx)
	if err != nil {
		return false, "", err
	}
	if !configured {
		return false, "模型未配置。请设置 MYCLAW_MODEL_PROVIDER、MYCLAW_MODEL_BASE_URL、MYCLAW_MODEL_API_KEY 和 MYCLAW_MODEL_NAME。", nil
	}
	return true, "模型已配置，可直接做文件总结和对话检索。", nil
}

func (a *DesktopApp) ingestFile(ctx context.Context, rawPath string) (knowledge.Entry, error) {
	input, ok, err := fileingest.Resolve(rawPath)
	if err != nil {
		return knowledge.Entry{}, err
	}
	if !ok {
		return knowledge.Entry{}, errors.New("只支持导入现有的图片或 PDF 文件。")
	}

	available, message, err := a.aiStatus(ctx)
	if err != nil {
		return knowledge.Entry{}, err
	}
	if !available {
		return knowledge.Entry{}, errors.New(message)
	}

	var summary string
	switch input.Kind {
	case fileingest.KindPDF:
		extractedText, extractErr := fileingest.ExtractPDFText(input.Path)
		if extractErr != nil {
			if errors.Is(extractErr, fileingest.ErrPDFExtractorUnavailable) {
				return knowledge.Entry{}, errors.New("当前这个构建不包含 PDF 文本提取能力。请使用启用 CGO 的构建来导入 PDF。")
			}
			return knowledge.Entry{}, extractErr
		}
		summary, err = a.aiService.SummarizePDFText(ctx, input.Name, extractedText)
	case fileingest.KindImage:
		summary, err = a.aiService.SummarizeImageFile(ctx, input.Name, input.DataURL)
	default:
		return knowledge.Entry{}, errors.New("暂不支持这个文件类型。")
	}
	if err != nil {
		return knowledge.Entry{}, err
	}

	return a.store.Add(ctx, knowledge.Entry{
		Text:       fileingest.FormatKnowledgeText(input, summary),
		Source:     desktopSourceLabel(),
		RecordedAt: time.Now(),
	})
}

func (a *DesktopApp) showReminderDialog(reminderItem reminder.Reminder) error {
	if a.ctx == nil {
		return nil
	}

	a.dialogMu.Lock()
	defer a.dialogMu.Unlock()

	runtime.EventsEmit(a.ctx, "reminder:due", map[string]string{
		"id":      reminderItem.ID,
		"shortId": shortID(reminderItem.ID),
		"message": reminderItem.Message,
	})
	runtime.WindowShow(a.ctx)
	_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "myclaw 提醒",
		Message: reminderItem.Message,
	})
	return err
}

func (a *DesktopApp) defaultDialogDirectory() string {
	candidates := []string{}
	if homeDir, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, homeDir)
	}
	candidates = append(candidates, a.dataDir)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func desktopMessageContext() appsvc.MessageContext {
	return appsvc.MessageContext{
		Interface: desktopInterface,
		UserID:    desktopUserID,
	}
}

func desktopSourceLabel() string {
	return desktopInterface + ":" + desktopUserID
}

func toKnowledgeItem(entry knowledge.Entry) KnowledgeItem {
	text := strings.TrimSpace(entry.Text)
	return KnowledgeItem{
		ID:             entry.ID,
		ShortID:        shortID(entry.ID),
		Text:           text,
		Preview:        preview(text, maxKnowledgePreviewRunes),
		Source:         strings.TrimSpace(entry.Source),
		RecordedAt:     entry.RecordedAt.Local().Format("2006-01-02 15:04:05"),
		RecordedAtUnix: entry.RecordedAt.Unix(),
		Keywords:       append([]string(nil), entry.Keywords...),
		IsFile:         strings.HasPrefix(text, "## 文件摘要"),
	}
}

func reverseKnowledge(entries []knowledge.Entry) {
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func preview(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}
