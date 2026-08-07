package models

const (
	ProviderGLM ModelProvider = "glm"

	GLM52    ModelID = "glm-5.2"
	GLM51    ModelID = "glm-5.1"
	GLM5     ModelID = "glm-5"
	GLM47    ModelID = "glm-4.7"
	GLM45Air ModelID = "glm-4.5-air"
)

var GLMModels = map[ModelID]Model{
	GLM52: {
		ID:                  GLM52,
		Name:                "GLM 5.2",
		Provider:            ProviderGLM,
		APIModel:            "glm-5.2",
		CostPer1MIn:         1.12,
		CostPer1MInCached:   0.28,
		CostPer1MOut:        3.92,
		CostPer1MOutCached:  3.92,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    8_000,
		CanReason:           false,
		SupportsAttachments: false,
	},

	GLM51: {
		ID:                  GLM51,
		Name:                "GLM 5.1",
		Provider:            ProviderGLM,
		APIModel:            "glm-5.1",
		CostPer1MIn:         0.84,
		CostPer1MInCached:   0.18,
		CostPer1MOut:        3.36,
		CostPer1MOutCached:  3.36,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8_000,
		CanReason:           false,
		SupportsAttachments: false,
	},

	GLM5: {
		ID:                  GLM5,
		Name:                "GLM 5",
		Provider:            ProviderGLM,
		APIModel:            "glm-5",
		CostPer1MIn:         0.56,
		CostPer1MInCached:   0.14,
		CostPer1MOut:        2.52,
		CostPer1MOutCached:  2.52,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8_000,
		CanReason:           false,
		SupportsAttachments: false,
	},

	GLM47: {
		ID:                  GLM47,
		Name:                "GLM 4.7",
		Provider:            ProviderGLM,
		APIModel:            "glm-4.7",
		CostPer1MIn:         0.28,
		CostPer1MInCached:   0.06,
		CostPer1MOut:        1.12,
		CostPer1MOutCached:  1.12,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8_000,
		CanReason:           false,
		SupportsAttachments: false,
	},

	GLM45Air: {
		ID:                  GLM45Air,
		Name:                "GLM 4.5 Air",
		Provider:            ProviderGLM,
		APIModel:            "glm-4.5-air",
		CostPer1MIn:         0.05,
		CostPer1MInCached:   0.05,
		CostPer1MOut:        0.15,
		CostPer1MOutCached:  0.15,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       128_000,
		DefaultMaxTokens:    8_000,
		CanReason:           false,
		SupportsAttachments: false,
	},
}
