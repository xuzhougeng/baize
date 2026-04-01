package main

import (
	"context"
	"fmt"
	goruntime "runtime"
	"sort"
	"strings"

	"myclaw/internal/ai"
	"myclaw/internal/dirlist"
	"myclaw/internal/filesearch"
	"myclaw/internal/systemcmd"
)

type ToolItem struct {
	Name            string `json:"name"`
	ShortName       string `json:"shortName"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Purpose         string `json:"purpose"`
	Provider        string `json:"provider"`
	ProviderKind    string `json:"providerKind"`
	SideEffectLevel string `json:"sideEffectLevel"`
	Status          string `json:"status"`
	StatusTone      string `json:"statusTone"`
	Configurable    bool   `json:"configurable"`
	ConfigValue     string `json:"configValue,omitempty"`
}

func (a *DesktopApp) ListTools() ([]ToolItem, error) {
	if a.service == nil {
		return nil, fmt.Errorf("工具服务尚未启用")
	}

	ctx := context.Background()
	project, err := a.currentProject(ctx)
	if err != nil {
		return nil, err
	}

	definitions, err := a.service.ListAgentToolDefinitions(ctx, desktopMessageContext(project, ""))
	if err != nil {
		return nil, err
	}

	settings, err := a.GetSettings()
	if err != nil {
		return nil, err
	}

	items := make([]ToolItem, 0, len(definitions))
	for _, definition := range definitions {
		items = append(items, toToolItem(definition, settings))
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := toolSortOrder(items[i].ShortName)
		right := toolSortOrder(items[j].ShortName)
		if left != right {
			return left < right
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})

	return items, nil
}

func toToolItem(definition ai.AgentToolDefinition, settings AppSettings) ToolItem {
	shortName := toolShortName(definition.Name)
	description := strings.TrimSpace(definition.Description)
	purpose := strings.TrimSpace(definition.Purpose)
	if description == "" {
		description = purpose
	}
	status, tone := toolStatus(definition.Name, settings)

	item := ToolItem{
		Name:            strings.TrimSpace(definition.Name),
		ShortName:       shortName,
		Title:           toolTitle(shortName),
		Description:     description,
		Purpose:         purpose,
		Provider:        strings.TrimSpace(definition.Provider),
		ProviderKind:    strings.TrimSpace(definition.ProviderKind),
		SideEffectLevel: strings.TrimSpace(definition.SideEffectLevel),
		Status:          status,
		StatusTone:      tone,
	}
	if shortName == filesearch.ToolName {
		item.Configurable = true
		item.ConfigValue = strings.TrimSpace(settings.WeixinEverythingPath)
	}
	return item
}

func toolShortName(name string) string {
	trimmed := strings.TrimSpace(name)
	if prefix, short, ok := strings.Cut(trimmed, "::"); ok && strings.TrimSpace(prefix) != "" {
		return strings.TrimSpace(short)
	}
	return trimmed
}

func toolTitle(name string) string {
	switch strings.TrimSpace(name) {
	case filesearch.ToolName:
		return "文件检索"
	case dirlist.ToolName:
		return "目录浏览"
	case systemcmd.ToolName:
		return "系统状态检查"
	case "knowledge_search":
		return "知识库检索"
	case "remember":
		return "保存知识"
	case "append_knowledge":
		return "补充知识"
	case "forget_knowledge":
		return "删除知识"
	case "reminder_list":
		return "查看提醒"
	case "reminder_add":
		return "创建提醒"
	case "reminder_remove":
		return "删除提醒"
	default:
		return strings.ReplaceAll(strings.TrimSpace(name), "_", " ")
	}
}

func toolSortOrder(name string) int {
	switch strings.TrimSpace(name) {
	case filesearch.ToolName:
		return 10
	case dirlist.ToolName:
		return 20
	case systemcmd.ToolName:
		return 30
	case "knowledge_search":
		return 40
	case "remember":
		return 50
	case "append_knowledge":
		return 60
	case "forget_knowledge":
		return 70
	case "reminder_list":
		return 80
	case "reminder_add":
		return 90
	case "reminder_remove":
		return 100
	default:
		return 999
	}
}

func toolStatus(name string, settings AppSettings) (string, string) {
	if toolShortName(name) != filesearch.ToolName {
		return "已就绪", "on"
	}
	if goruntime.GOOS != "windows" {
		return "当前平台暂不支持", "off"
	}
	if strings.TrimSpace(settings.WeixinEverythingPath) == "" {
		return "需配置 es.exe 路径", "pending"
	}
	return "已就绪", "on"
}
