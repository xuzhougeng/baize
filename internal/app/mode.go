package app

import "strings"

type Mode string

const (
	ModeAsk               Mode = "ask"
	modeKnowledgeOverride Mode = "knowledge"
	ModeAgent             Mode = "agent"
)

func normalizeMode(value string) Mode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ModeAsk), "direct", "ai", "@ai", string(modeKnowledgeOverride):
		return ModeAsk
	case string(ModeAgent), "@agent":
		return ModeAgent
	default:
		return ""
	}
}

func defaultMode() Mode {
	return ModeAgent
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
		{prefix: "@ai", mode: ModeAsk},
		{prefix: "@kb", mode: modeKnowledgeOverride},
		{prefix: "@agent", mode: ModeAgent},
	} {
		if !strings.HasPrefix(lower, candidate.prefix) {
			continue
		}
		return candidate.mode, strings.TrimSpace(text[len(candidate.prefix):]), true
	}
	return "", input, false
}
