package modelconfig

import (
	"strings"
	"time"
)

func normalizeOptionalPositiveInt(value *int) *int {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return nil
	}
	return value
}

func normalizeDatabase(db *databaseFile) bool {
	if db == nil {
		return false
	}
	changed := false
	if db.Version != currentDatabaseVersion {
		db.Version = currentDatabaseVersion
		changed = true
	}
	for index := range db.Profiles {
		if normalizeStoredProfile(&db.Profiles[index]) {
			changed = true
		}
	}
	return changed
}

func normalizeStoredProfile(profile *storedProfile) bool {
	if profile == nil {
		return false
	}
	before := *profile
	cfg := profile.config().Normalize()
	profile.ID = strings.TrimSpace(profile.ID)
	if profile.ID == "" {
		profile.ID = newID()
	}
	profile.Name = cfg.Name
	profile.Provider = cfg.Provider
	profile.APIType = cfg.APIType
	profile.BaseURL = cfg.BaseURL
	profile.Model = cfg.Model
	profile.MaxOutputTokensText = cfg.MaxOutputTokensText
	profile.MaxOutputTokensJSON = cfg.MaxOutputTokensJSON
	profile.MaxOutputTokens = nil
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now().UTC()
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = profile.CreatedAt
	}
	return before != *profile
}

func repairActiveProfile(db *databaseFile) bool {
	if db == nil {
		return false
	}
	if len(db.Profiles) == 0 {
		if db.ActiveProfileID != "" {
			db.ActiveProfileID = ""
			return true
		}
		return false
	}
	if indexOfProfile(db.Profiles, db.ActiveProfileID) != -1 {
		return false
	}
	db.ActiveProfileID = db.Profiles[0].ID
	return true
}

func normalizeProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", ProviderOpenAI:
		return ProviderOpenAI
	case ProviderAnthropic, "antrophic":
		return ProviderAnthropic
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeAPIType(provider, value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return defaultAPIType(provider)
	case "response":
		return APITypeResponses
	case "responses":
		return APITypeResponses
	case "chat_completion", "chat-completion", "chat completions", "chat completion", "chat-completions":
		return APITypeChatCompletions
	case APITypeChatCompletions:
		return APITypeChatCompletions
	case "message":
		return APITypeMessages
	case APITypeMessages:
		return APITypeMessages
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func defaultAPIType(provider string) string {
	switch normalizeProvider(provider) {
	case ProviderAnthropic:
		return APITypeMessages
	default:
		return APITypeResponses
	}
}

func defaultBaseURL(provider string) string {
	switch normalizeProvider(provider) {
	case ProviderAnthropic:
		return DefaultAnthropicBaseURL
	default:
		return DefaultBaseURL
	}
}

func normalizeProfileName(name, provider, apiType, model string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	model = strings.TrimSpace(model)
	if model != "" {
		return model
	}
	label := providerLabel(normalizeProvider(provider))
	switch normalizeAPIType(provider, apiType) {
	case APITypeResponses:
		return label + " Responses"
	case APITypeChatCompletions:
		return label + " Chat Completions"
	case APITypeMessages:
		return label + " Messages"
	default:
		return label
	}
}

func providerLabel(provider string) string {
	switch normalizeProvider(provider) {
	case ProviderAnthropic:
		return "Anthropic"
	default:
		return "OpenAI"
	}
}
