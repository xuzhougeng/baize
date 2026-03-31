package main

import (
	"fmt"
	"strings"

	"myclaw/internal/knowledge"
	"myclaw/internal/modelconfig"
	"myclaw/internal/promptlib"
	"myclaw/internal/reminder"
	"myclaw/internal/skilllib"
)

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

func toPromptItem(prompt promptlib.Prompt) PromptItem {
	content := strings.TrimSpace(prompt.Content)
	return PromptItem{
		ID:             prompt.ID,
		ShortID:        shortID(prompt.ID),
		Title:          strings.TrimSpace(prompt.Title),
		Content:        content,
		Preview:        preview(content, maxKnowledgePreviewRunes),
		RecordedAt:     prompt.RecordedAt.Local().Format("2006-01-02 15:04:05"),
		RecordedAtUnix: prompt.RecordedAt.Unix(),
	}
}

func toReminderItem(item reminder.Reminder) ReminderItem {
	frequency := string(item.Frequency)
	frequencyLabel := "单次"
	scheduleLabel := "单次"
	if item.Frequency == reminder.FrequencyDaily {
		frequencyLabel = "每天"
		scheduleLabel = fmt.Sprintf("每天 %02d:%02d", item.DailyHour, item.DailyMinute)
	}

	return ReminderItem{
		ID:             item.ID,
		ShortID:        shortID(item.ID),
		Message:        strings.TrimSpace(item.Message),
		Frequency:      frequency,
		FrequencyLabel: frequencyLabel,
		ScheduleLabel:  scheduleLabel,
		NextRunAt:      item.NextRunAt.Local().Format("2006-01-02 15:04:05"),
		NextRunAtUnix:  item.NextRunAt.Unix(),
		CreatedAt:      item.CreatedAt.Local().Format("2006-01-02 15:04:05"),
		CreatedAtUnix:  item.CreatedAt.Unix(),
	}
}

func toSkillItem(skill skilllib.Skill, loaded bool) SkillItem {
	return SkillItem{
		Name:        strings.TrimSpace(skill.Name),
		Description: strings.TrimSpace(skill.Description),
		Content:     strings.TrimSpace(skill.Content),
		Dir:         strings.TrimSpace(skill.Dir),
		Loaded:      loaded,
	}
}

func toProjectSummary(info knowledge.ProjectInfo, activeProject string) ProjectSummary {
	return ProjectSummary{
		Name:                 knowledge.CanonicalProjectName(info.Name),
		KnowledgeCount:       info.KnowledgeCount,
		LatestRecordedAt:     info.LatestRecordedAt.Local().Format("2006-01-02 15:04:05"),
		LatestRecordedAtUnix: info.LatestRecordedAt.Unix(),
		Active:               strings.EqualFold(knowledge.CanonicalProjectName(info.Name), knowledge.CanonicalProjectName(activeProject)),
	}
}

func reverseKnowledge(entries []knowledge.Entry) {
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
}

func reversePrompts(prompts []promptlib.Prompt) {
	for left, right := 0, len(prompts)-1; left < right; left, right = left+1, right-1 {
		prompts[left], prompts[right] = prompts[right], prompts[left]
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

func desktopModelMessage(snapshot modelconfig.Snapshot, missing []string) string {
	switch {
	case len(snapshot.Profiles) == 0:
		return "尚未保存任何模型 profile。"
	case snapshot.ActiveProfileID == "":
		return "已保存模型 profile，但尚未选择活跃模型。"
	case len(missing) == 0:
		return fmt.Sprintf("本地已保存 %d 个模型 profile，当前活跃模型已生效。", len(snapshot.Profiles))
	default:
		return "当前活跃模型 profile 已保存，但配置仍不完整。"
	}
}
