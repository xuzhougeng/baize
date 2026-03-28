package app

import "strings"

type Mode string

const (
	ModeDirect    Mode = "direct"
	ModeKnowledge Mode = "knowledge"
	ModeAgent     Mode = "agent"
)

func normalizeMode(value string) Mode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ModeDirect), "ai", "@ai":
		return ModeDirect
	case string(ModeKnowledge), "kb", "@kb":
		return ModeKnowledge
	case string(ModeAgent), "@agent":
		return ModeAgent
	default:
		return ""
	}
}

func defaultMode() Mode {
	return ModeDirect
}

func parseModeOverride(input string) (Mode, string, bool) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", "", false
	}

	lower := strings.ToLower(text)
	for _, candidate := range []struct {
		prefix string
		mode   Mode
	}{
		{prefix: "@ai", mode: ModeDirect},
		{prefix: "@kb", mode: ModeKnowledge},
		{prefix: "@agent", mode: ModeAgent},
	} {
		if !strings.HasPrefix(lower, candidate.prefix) {
			continue
		}
		return candidate.mode, strings.TrimSpace(text[len(candidate.prefix):]), true
	}
	return "", input, false
}

func modeUsage() string {
	return "用法:\n" +
		"/mode\n" +
		"/mode direct\n" +
		"/mode knowledge\n" +
		"/mode agent\n\n" +
		"也可以在单条消息前加 `@ai`、`@kb` 或 `@agent` 临时覆盖当前模式。"
}

func modeDescription(mode Mode) string {
	switch mode {
	case ModeDirect:
		return "普通问题直接走 AI，不依赖知识库。"
	case ModeKnowledge:
		return "普通问题走知识库检索和候选复核。"
	case ModeAgent:
		return "预留给未来的工具执行模式。"
	default:
		return ""
	}
}
