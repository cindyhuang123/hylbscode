package models

import "fmt"

// validProviders lists the providers that NewProvider can route to a client.
// Extra models must reference one of these.
var validProviders = map[ModelProvider]bool{
	ProviderCopilot:    true,
	ProviderAnthropic:  true,
	ProviderOpenAI:     true,
	ProviderGemini:     true,
	ProviderBedrock:    true,
	ProviderGROQ:       true,
	ProviderAzure:      true,
	ProviderVertexAI:   true,
	ProviderOpenRouter: true,
	ProviderXAI:        true,
	ProviderDeepSeek:   true,
	ProviderGLM:        true,
	ProviderLocal:      true,
}

// MergeExtraModels registers user-defined models from the config file into
// SupportedModels. A model whose ID already exists (built-in) replaces the
// built-in definition entirely, so pricing and other metadata can be
// overridden without recompiling. Provider-specific capability flags such as
// CanReason and SupportsAttachments must match what the provider client
// actually supports.
func MergeExtraModels(extras []Model) error {
	for _, m := range extras {
		if m.ID == "" {
			return fmt.Errorf("extra model: id is required")
		}
		if m.APIModel == "" {
			return fmt.Errorf("extra model %q: apiModel is required", m.ID)
		}
		if m.Provider == "" {
			return fmt.Errorf("extra model %q: provider is required", m.ID)
		}
		if !validProviders[m.Provider] {
			return fmt.Errorf("extra model %q: unsupported provider %q", m.ID, m.Provider)
		}
		if m.ContextWindow <= 0 {
			return fmt.Errorf("extra model %q: contextWindow must be greater than 0", m.ID)
		}
		SupportedModels[m.ID] = m
	}
	return nil
}
