package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/llm/tools"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/message"
)

type EventType string

const maxRetries = 8

const (
	EventContentStart  EventType = "content_start"
	EventToolUseStart  EventType = "tool_use_start"
	EventToolUseDelta  EventType = "tool_use_delta"
	EventToolUseStop   EventType = "tool_use_stop"
	EventContentDelta  EventType = "content_delta"
	EventThinkingDelta EventType = "thinking_delta"
	EventContentStop   EventType = "content_stop"
	EventComplete      EventType = "complete"
	EventError         EventType = "error"
	EventWarning       EventType = "warning"
)

type TokenUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type ProviderResponse struct {
	Content      string
	ToolCalls    []message.ToolCall
	Usage        TokenUsage
	FinishReason message.FinishReason
}

type ProviderEvent struct {
	Type EventType

	Content  string
	Thinking string
	Response *ProviderResponse
	ToolCall *message.ToolCall
	Error    error
}
type Provider interface {
	SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)

	StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent

	Model() models.Model
}

type providerClientOptions struct {
	apiKey        string
	model         models.Model
	maxTokens     int64
	systemMessage string

	anthropicOptions []AnthropicOption
	openaiOptions    []OpenAIOption
	geminiOptions    []GeminiOption
	bedrockOptions   []BedrockOption
	copilotOptions   []CopilotOption
}

type ProviderClientOption func(*providerClientOptions)

type ProviderClient interface {
	send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)
	stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
}

type baseProvider[C ProviderClient] struct {
	options providerClientOptions
	client  C
}

// defaultOpenAIBaseURLs holds the standard OpenAI-compatible endpoint for each
// provider that reuses the OpenAI client. An empty value means the SDK default.
var defaultOpenAIBaseURLs = map[models.ModelProvider]string{
	models.ProviderOpenAI:     "https://api.openai.com/v1",
	models.ProviderGROQ:       "https://api.groq.com/openai/v1",
	models.ProviderOpenRouter: "https://openrouter.ai/api/v1",
	models.ProviderXAI:        "https://api.x.ai/v1",
	models.ProviderDeepSeek:   "https://api.deepseek.com",
	models.ProviderGLM:        "https://open.bigmodel.cn/api/paas/v4",
}

// DefaultBaseURL returns the standard OpenAI-compatible base URL for a
// provider, or "" when the provider uses its SDK default endpoint.
func DefaultBaseURL(provider models.ModelProvider) string {
	return defaultOpenAIBaseURLs[provider]
}

// openAIBaseURL resolves the base URL for an OpenAI-compatible provider,
// preferring the per-provider override stored in the config file (e.g. an
// OpenAI-compatible third-party endpoint) over the provider's default.
func openAIBaseURL(provider models.ModelProvider) string {
	if cfg := config.Get(); cfg != nil {
		if u := strings.TrimSpace(cfg.Providers[provider].BaseURL); u != "" {
			return u
		}
	}
	return DefaultBaseURL(provider)
}

func NewProvider(providerName models.ModelProvider, opts ...ProviderClientOption) (Provider, error) {
	clientOptions := providerClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}
	switch providerName {
	case models.ProviderCopilot:
		return &baseProvider[CopilotClient]{
			options: clientOptions,
			client:  newCopilotClient(clientOptions),
		}, nil
	case models.ProviderAnthropic:
		return &baseProvider[AnthropicClient]{
			options: clientOptions,
			client:  newAnthropicClient(clientOptions),
		}, nil
	case models.ProviderOpenAI:
		if u := openAIBaseURL(models.ProviderOpenAI); u != "" {
			clientOptions.openaiOptions = append(clientOptions.openaiOptions,
				WithOpenAIBaseURL(u),
			)
		}
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderGemini:
		return &baseProvider[GeminiClient]{
			options: clientOptions,
			client:  newGeminiClient(clientOptions),
		}, nil
	case models.ProviderBedrock:
		return &baseProvider[BedrockClient]{
			options: clientOptions,
			client:  newBedrockClient(clientOptions),
		}, nil
	case models.ProviderGROQ:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(openAIBaseURL(models.ProviderGROQ)),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderAzure:
		return &baseProvider[AzureClient]{
			options: clientOptions,
			client:  newAzureClient(clientOptions),
		}, nil
	case models.ProviderVertexAI:
		return &baseProvider[VertexAIClient]{
			options: clientOptions,
			client:  newVertexAIClient(clientOptions),
		}, nil
	case models.ProviderOpenRouter:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(openAIBaseURL(models.ProviderOpenRouter)),
			WithOpenAIExtraHeaders(map[string]string{
				"HTTP-Referer": "hylbscode.ai",
				"X-Title":      "HyLbsCode",
			}),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderXAI:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(openAIBaseURL(models.ProviderXAI)),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderDeepSeek:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(openAIBaseURL(models.ProviderDeepSeek)),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderGLM:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(openAIBaseURL(models.ProviderGLM)),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderLocal:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(os.Getenv("LOCAL_ENDPOINT")),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil
	case models.ProviderMock:
		// TODO: implement mock client for test
		panic("not implemented")
	}
	return nil, fmt.Errorf("provider not supported: %s", providerName)
}

func (p *baseProvider[C]) cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		// The message has no content
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func (p *baseProvider[C]) SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	messages = p.cleanMessages(messages)
	logRequestToLLM(ctx, p.options, messages, len(tools))
	resp, err := p.client.send(ctx, messages, tools)
	logResponseFromLLM(ctx, p.options, resp, err)
	return resp, err
}

func (p *baseProvider[C]) Model() models.Model {
	return p.options.model
}

func (p *baseProvider[C]) StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	messages = p.cleanMessages(messages)
	logRequestToLLM(ctx, p.options, messages, len(tools))
	in := p.client.stream(ctx, messages, tools)
	out := make(chan ProviderEvent)
	go func() {
		defer close(out)
		for ev := range in {
			switch ev.Type {
			case EventComplete:
				logResponseFromLLM(ctx, p.options, ev.Response, nil)
			case EventError:
				logResponseFromLLM(ctx, p.options, nil, ev.Error)
			}
			out <- ev
		}
	}()
	return out
}

func sessionID(ctx context.Context) string {
	sid, _ := ctx.Value(tools.SessionIDContextKey).(string)
	return sid
}

func logRequestToLLM(ctx context.Context, p providerClientOptions, messages []message.Message, toolCount int) {
	data, _ := json.Marshal(messages)
	logging.Info("请求to llm",
		"provider", p.model.Provider,
		"model", p.model.ID,
		"messages", string(data),
		"tools", toolCount,
		"session_id", sessionID(ctx),
	)
}

func logResponseFromLLM(ctx context.Context, p providerClientOptions, resp *ProviderResponse, err error) {
	if err != nil {
		logging.Info("响应from llm", "provider", p.model.Provider, "model", p.model.ID, "error", err)
		return
	}
	toolCalls, _ := json.Marshal(resp.ToolCalls)
	logging.Info("响应from llm",
		"provider", p.model.Provider,
		"model", p.model.ID,
		"CONTENT", resp.Content,
		"TOOL_CALLS", string(toolCalls),
		"USAGE", fmt.Sprintf("in:%d out:%d", resp.Usage.InputTokens, resp.Usage.OutputTokens),
		"session_id", sessionID(ctx),
	)
}

func WithAPIKey(apiKey string) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.apiKey = apiKey
	}
}

func WithModel(model models.Model) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.model = model
	}
}

func WithMaxTokens(maxTokens int64) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.maxTokens = maxTokens
	}
}

func WithSystemMessage(systemMessage string) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.systemMessage = systemMessage
	}
}

func WithAnthropicOptions(anthropicOptions ...AnthropicOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.anthropicOptions = anthropicOptions
	}
}

func WithOpenAIOptions(openaiOptions ...OpenAIOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

func WithGeminiOptions(geminiOptions ...GeminiOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.geminiOptions = geminiOptions
	}
}

func WithBedrockOptions(bedrockOptions ...BedrockOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.bedrockOptions = bedrockOptions
	}
}

func WithCopilotOptions(copilotOptions ...CopilotOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.copilotOptions = copilotOptions
	}
}
