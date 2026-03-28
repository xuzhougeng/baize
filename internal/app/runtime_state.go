package app

import (
	"context"
	"slices"
	"strings"

	"myclaw/internal/ai"
	"myclaw/internal/sessionstate"
)

const (
	maxConversationHistoryMessages = 12
	maxConversationHistoryRunes    = 360
)

func (s *Service) sessionSnapshot(ctx context.Context, mc MessageContext) (sessionstate.Snapshot, error) {
	return s.sessionSnapshotByKey(ctx, conversationSessionKey(mc))
}

func (s *Service) sessionSnapshotByKey(ctx context.Context, key string) (sessionstate.Snapshot, error) {
	snapshot := sessionstate.Snapshot{Key: strings.TrimSpace(key)}
	if s.sessionStore == nil || snapshot.Key == "" {
		return snapshot, nil
	}

	saved, ok, err := s.sessionStore.Load(ctx, snapshot.Key)
	if err != nil {
		return sessionstate.Snapshot{}, err
	}
	if ok {
		return saved, nil
	}
	return snapshot, nil
}

func (s *Service) saveSessionSnapshot(ctx context.Context, snapshot sessionstate.Snapshot) error {
	if s.sessionStore == nil {
		return nil
	}
	snapshot.Key = strings.TrimSpace(snapshot.Key)
	if snapshot.Key == "" {
		return nil
	}
	_, err := s.sessionStore.Save(ctx, snapshot)
	return err
}

func (s *Service) conversationHistory(ctx context.Context, mc MessageContext) []ai.ConversationMessage {
	snapshot, err := s.sessionSnapshot(ctx, mc)
	if err != nil {
		return nil
	}

	history := make([]ai.ConversationMessage, 0, len(snapshot.History))
	for _, item := range snapshot.History {
		history = append(history, ai.ConversationMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}
	return trimConversationHistory(history)
}

func (s *Service) appendConversationHistory(ctx context.Context, mc MessageContext, userInput, assistantReply string) {
	snapshot, err := s.sessionSnapshot(ctx, mc)
	if err != nil {
		return
	}

	history := append([]sessionstate.Message(nil), snapshot.History...)
	history = append(history,
		sessionstate.Message{
			Role:    "user",
			Content: trimConversationHistoryText(userInput),
		},
		sessionstate.Message{
			Role:    "assistant",
			Content: trimConversationHistoryText(assistantReply),
		},
	)
	snapshot.History = trimSessionHistory(history)
	_ = s.saveSessionSnapshot(ctx, snapshot)
}

func (s *Service) persistedLoadedSkillNames(mc MessageContext) []string {
	snapshot, err := s.sessionSnapshot(context.Background(), mc)
	if err != nil {
		return nil
	}
	out := append([]string(nil), snapshot.LoadedSkills...)
	slices.Sort(out)
	return out
}

func (s *Service) setPersistedLoadedSkillNames(mc MessageContext, names []string) {
	snapshot, err := s.sessionSnapshot(context.Background(), mc)
	if err != nil {
		return
	}
	snapshot.LoadedSkills = normalizeStringList(names)
	_ = s.saveSessionSnapshot(context.Background(), snapshot)
}

func (s *Service) selectedPromptID(ctx context.Context, mc MessageContext) string {
	snapshot, err := s.sessionSnapshot(ctx, mc)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(snapshot.PromptID)
}

func (s *Service) setSelectedPromptID(ctx context.Context, mc MessageContext, promptID string) error {
	snapshot, err := s.sessionSnapshot(ctx, mc)
	if err != nil {
		return err
	}
	snapshot.PromptID = strings.TrimSpace(promptID)
	return s.saveSessionSnapshot(ctx, snapshot)
}

func trimConversationHistory(history []ai.ConversationMessage) []ai.ConversationMessage {
	history = ai.NormalizeConversationMessages(history)
	if len(history) <= maxConversationHistoryMessages {
		return history
	}
	return history[len(history)-maxConversationHistoryMessages:]
}

func trimSessionHistory(history []sessionstate.Message) []sessionstate.Message {
	out := make([]sessionstate.Message, 0, len(history))
	for _, item := range history {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := trimConversationHistoryText(item.Content)
		if content == "" {
			continue
		}
		out = append(out, sessionstate.Message{
			Role:    role,
			Content: content,
		})
	}
	if len(out) <= maxConversationHistoryMessages {
		return out
	}
	return out[len(out)-maxConversationHistoryMessages:]
}

func trimConversationHistoryText(text string) string {
	return preview(strings.TrimSpace(text), maxConversationHistoryRunes)
}

func normalizeStringList(values []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}
