package models

const (
	ProviderDeepSeek ModelProvider = "deepseek"

	DeepSeekV4Flash ModelID = "deepseek-v4-flash"
	DeepSeekV4Pro   ModelID = "deepseek-v4-pro"
)

var DeepSeekModels = map[ModelID]Model{
	DeepSeekV4Flash: {
		ID:                  DeepSeekV4Flash,
		Name:                "DeepSeek V4 Flash",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-v4-flash",
		CostPer1MIn:         0.14,
		CostPer1MInCached:   0.0028,
		CostPer1MOut:        0.28,
		CostPer1MOutCached:  0.0028,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    8_000,
		CanReason:           true,
		SupportsAttachments: false,
	},

	DeepSeekV4Pro: {
		ID:                  DeepSeekV4Pro,
		Name:                "DeepSeek V4 Pro",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-v4-pro",
		CostPer1MIn:         0.435,
		CostPer1MInCached:   0.003625,
		CostPer1MOut:        0.87,
		CostPer1MOutCached:  0.003625,
		CostCurrency:        CurrencyCNY,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    8_000,
		CanReason:           true,
		SupportsAttachments: false,
	},
}
