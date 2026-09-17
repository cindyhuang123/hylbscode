package provider

import (
	"testing"

	"github.com/openai/openai-go"
)

func TestUsageExtractsDeepSeekCacheHits(t *testing.T) {
	cc := openai.ChatCompletion{
		Usage: openai.CompletionUsage{
			PromptTokens:     12000,
			CompletionTokens: 500,
			TotalTokens:      12500,
		},
	}
	// The SDK's stream accumulator copies usage numbers but drops the chunk's
	// raw JSON; simulate that shape and expect no cache tokens.
	usage := (&openaiClient{}).usage(cc, "")
	if usage.CacheReadTokens != 0 {
		t.Fatalf("stream path without raw JSON: CacheReadTokens = %d, want 0", usage.CacheReadTokens)
	}

	usageRawJSON := `{"prompt_tokens":12000,"prompt_cache_hit_tokens":9000,"prompt_cache_miss_tokens":3000,"completion_tokens":500}`
	usage = (&openaiClient{}).usage(cc, usageRawJSON)
	if usage.CacheReadTokens != 9000 {
		t.Fatalf("stream path with raw usage JSON: CacheReadTokens = %d, want 9000", usage.CacheReadTokens)
	}
}
