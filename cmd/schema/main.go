package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
)

// JSONSchemaType represents a JSON Schema type
type JSONSchemaType struct {
	Type                 string           `json:"type,omitempty"`
	Description          string           `json:"description,omitempty"`
	Properties           map[string]any   `json:"properties,omitempty"`
	Required             []string         `json:"required,omitempty"`
	AdditionalProperties any              `json:"additionalProperties,omitempty"`
	Enum                 []any            `json:"enum,omitempty"`
	Items                map[string]any   `json:"items,omitempty"`
	OneOf                []map[string]any `json:"oneOf,omitempty"`
	AnyOf                []map[string]any `json:"anyOf,omitempty"`
	Default              any              `json:"default,omitempty"`
}

func main() {
	schema := generateSchema()

	// Pretty print the schema
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding schema: %v\n", err)
		os.Exit(1)
	}
}

func generateSchema() map[string]any {
	schema := map[string]any{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       "HyLbsCode Configuration",
		"description": "Configuration schema for the HyLbsCode application",
		"type":        "object",
		"properties":  map[string]any{},
	}

	// Add Data configuration
	schema["properties"].(map[string]any)["data"] = map[string]any{
		"type":        "object",
		"description": "Storage configuration",
		"properties": map[string]any{
			"directory": map[string]any{
				"type":        "string",
				"description": "Directory where application data is stored",
				"default":     ".hylbscode",
			},
		},
		"required": []string{"directory"},
	}

	// Add working directory
	schema["properties"].(map[string]any)["wd"] = map[string]any{
		"type":        "string",
		"description": "Working directory for the application",
	}

	// Add debug flags
	schema["properties"].(map[string]any)["debug"] = map[string]any{
		"type":        "boolean",
		"description": "Enable debug mode",
		"default":     false,
	}

	schema["properties"].(map[string]any)["debugLSP"] = map[string]any{
		"type":        "boolean",
		"description": "Enable LSP debug mode",
		"default":     false,
	}

	schema["properties"].(map[string]any)["autoCompact"] = map[string]any{
		"type":        "boolean",
		"description": "Automatically summarize the session when approaching the context window limit",
		"default":     true,
	}

	schema["properties"].(map[string]any)["autoCompactThreshold"] = map[string]any{
		"type":        "number",
		"description": "Fraction of the context window at which automatic compaction triggers (0-1)",
		"default":     0.7,
		"minimum":     0,
		"maximum":     1,
	}

	schema["properties"].(map[string]any)["contextPaths"] = map[string]any{
		"type":        "array",
		"description": "Context paths for the application",
		"items": map[string]any{
			"type": "string",
		},
		"default": []string{
			".github/copilot-instructions.md",
			".cursorrules",
			".cursor/rules/",
			"CLAUDE.md",
			"CLAUDE.local.md",
			"hylbscode.md",
			"hylbscode.local.md",
			"HyLbsCode.md",
			"HyLbsCode.local.md",
			"HYLBSCODE.md",
			"HYLBSCODE.local.md",
		},
	}

	schema["properties"].(map[string]any)["tui"] = map[string]any{
		"type":        "object",
		"description": "Terminal User Interface configuration",
		"properties": map[string]any{
			"theme": map[string]any{
				"type":        "string",
				"description": "TUI theme name",
				"default":     "hylbscode",
				"enum": []string{
					"hylbscode",
					"catppuccin",
					"dracula",
					"flexoki",
					"gruvbox",
					"monokai",
					"onedark",
					"tron",
				},
			},
			"language": map[string]any{
				"type":        "string",
				"description": "TUI language",
				"default":     "en",
				"enum": []string{
					"en",
					"zh",
				},
			},
		},
	}

	// Add GUI configuration
	schema["properties"].(map[string]any)["gui"] = map[string]any{
		"type":        "object",
		"description": "Fyne desktop interface configuration",
		"properties": map[string]any{
			"theme": map[string]any{
				"type":        "string",
				"description": "GUI theme: auto, light, or dark",
				"default":     "auto",
				"enum": []string{
					"auto",
					"light",
					"dark",
				},
			},
			"width": map[string]any{
				"type":        "integer",
				"description": "Main window width in pixels",
			},
			"height": map[string]any{
				"type":        "integer",
				"description": "Main window height in pixels",
			},
			"cnyRate": map[string]any{
				"type":        "number",
				"description": "USD to CNY exchange rate used for cost display",
			},
			"font": map[string]any{
				"type":        "string",
				"description": "Path to a ttf/otf font file that overrides the built-in UI font",
			},
		},
	}

	// Add MCP servers
	schema["properties"].(map[string]any)["mcpServers"] = map[string]any{
		"type":        "object",
		"description": "Model Control Protocol server configurations",
		"additionalProperties": map[string]any{
			"type":        "object",
			"description": "MCP server configuration",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Command to execute for the MCP server",
				},
				"env": map[string]any{
					"type":        "array",
					"description": "Environment variables for the MCP server",
					"items": map[string]any{
						"type": "string",
					},
				},
				"args": map[string]any{
					"type":        "array",
					"description": "Command arguments for the MCP server",
					"items": map[string]any{
						"type": "string",
					},
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Type of MCP server",
					"enum":        []string{"stdio", "sse"},
					"default":     "stdio",
				},
				"url": map[string]any{
					"type":        "string",
					"description": "URL for SSE type MCP servers",
				},
				"headers": map[string]any{
					"type":        "object",
					"description": "HTTP headers for SSE type MCP servers",
					"additionalProperties": map[string]any{
						"type": "string",
					},
				},
			},
			"required": []string{"command"},
		},
	}

	// Add providers
	providerSchema := map[string]any{
		"type":        "object",
		"description": "LLM provider configurations",
		"additionalProperties": map[string]any{
			"type":        "object",
			"description": "Provider configuration",
			"properties": map[string]any{
				"apiKey": map[string]any{
					"type":        "string",
					"description": "API key for the provider",
				},
				"disabled": map[string]any{
					"type":        "boolean",
					"description": "Whether the provider is disabled",
					"default":     false,
				},
			},
		},
	}

	// Add known providers
	knownProviders := []string{
		string(models.ProviderCopilot),
		string(models.ProviderAnthropic),
		string(models.ProviderOpenAI),
		string(models.ProviderGemini),
		string(models.ProviderBedrock),
		string(models.ProviderGROQ),
		string(models.ProviderAzure),
		string(models.ProviderVertexAI),
		string(models.ProviderOpenRouter),
		string(models.ProviderXAI),
		string(models.ProviderDeepSeek),
		string(models.ProviderGLM),
		string(models.ProviderLocal),
	}

	providerSchema["additionalProperties"].(map[string]any)["properties"].(map[string]any)["provider"] = map[string]any{
		"type":        "string",
		"description": "Provider type",
		"enum":        knownProviders,
	}

	schema["properties"].(map[string]any)["providers"] = providerSchema

	// Add extra models
	extraModelProps := map[string]any{
		"id": map[string]any{
			"type":        "string",
			"description": "Unique model ID, referenced by agents.*.model. Replaces the built-in model with the same ID.",
		},
		"name": map[string]any{
			"type":        "string",
			"description": "Display name",
		},
		"provider": map[string]any{
			"type":        "string",
			"description": "Provider serving this model; must have an API key configured",
			"enum":        knownProviders,
		},
		"apiModel": map[string]any{
			"type":        "string",
			"description": "Model name sent to the provider API",
		},
		"costPer1MIn": map[string]any{
			"type":        "number",
			"description": "Input cost per 1M tokens (USD)",
			"minimum":     0,
		},
		"costPer1MOut": map[string]any{
			"type":        "number",
			"description": "Output cost per 1M tokens (USD)",
			"minimum":     0,
		},
		"costPer1MInCached": map[string]any{
			"type":        "number",
			"description": "Cached input cost per 1M tokens (USD)",
			"minimum":     0,
		},
		"costPer1MOutCached": map[string]any{
			"type":        "number",
			"description": "Cached output cost per 1M tokens (USD)",
			"minimum":     0,
		},
		"contextWindow": map[string]any{
			"type":        "integer",
			"description": "Context window size in tokens",
			"minimum":     1,
		},
		"defaultMaxTokens": map[string]any{
			"type":        "integer",
			"description": "Default maximum output tokens",
			"minimum":     1,
		},
		"canReason": map[string]any{
			"type":        "boolean",
			"description": "Whether the model supports reasoning effort. Must match what the provider client supports, as it changes the request format.",
		},
		"supportsAttachments": map[string]any{
			"type":        "boolean",
			"description": "Whether the model accepts image attachments",
		},
	}

	schema["properties"].(map[string]any)["extraModels"] = map[string]any{
		"type":        "array",
		"description": "Register additional models from the config file without recompiling",
		"items": map[string]any{
			"type":       "object",
			"properties": extraModelProps,
			"required":   []string{"id", "provider", "apiModel", "contextWindow"},
		},
	}

	// Add agents
	agentSchema := map[string]any{
		"type":        "object",
		"description": "Agent configurations",
		"additionalProperties": map[string]any{
			"type":        "object",
			"description": "Agent configuration",
			"properties": map[string]any{
				"model": map[string]any{
					"type":        "string",
					"description": "Model ID for the agent",
				},
				"maxTokens": map[string]any{
					"type":        "integer",
					"description": "Maximum tokens for the agent",
					"minimum":     1,
				},
				"reasoningEffort": map[string]any{
					"type":        "string",
					"description": "Reasoning effort for models that support it (OpenAI, Anthropic)",
					"enum":        []string{"low", "medium", "high"},
				},
			},
			"required": []string{"model"},
		},
	}

	// Add model enum
	modelEnum := []string{}
	for modelID := range models.SupportedModels {
		modelEnum = append(modelEnum, string(modelID))
	}
	agentSchema["additionalProperties"].(map[string]any)["properties"].(map[string]any)["model"].(map[string]any)["enum"] = modelEnum

	// Add specific agent properties
	agentProperties := map[string]any{}
	knownAgents := []string{
		string(config.AgentCoder),
		string(config.AgentTask),
		string(config.AgentTitle),
	}

	for _, agentName := range knownAgents {
		agentProperties[agentName] = map[string]any{
			"$ref": "#/definitions/agent",
		}
	}

	// Create a combined schema that allows both specific agents and additional ones
	combinedAgentSchema := map[string]any{
		"type":                 "object",
		"description":          "Agent configurations",
		"properties":           agentProperties,
		"additionalProperties": agentSchema["additionalProperties"],
	}

	schema["properties"].(map[string]any)["agents"] = combinedAgentSchema
	schema["definitions"] = map[string]any{
		"agent": agentSchema["additionalProperties"],
	}

	// Add LSP configuration
	schema["properties"].(map[string]any)["lsp"] = map[string]any{
		"type":        "object",
		"description": "Language Server Protocol configurations",
		"additionalProperties": map[string]any{
			"type":        "object",
			"description": "LSP configuration for a language",
			"properties": map[string]any{
				"disabled": map[string]any{
					"type":        "boolean",
					"description": "Whether the LSP is disabled",
					"default":     false,
				},
				"command": map[string]any{
					"type":        "string",
					"description": "Command to execute for the LSP server",
				},
				"args": map[string]any{
					"type":        "array",
					"description": "Command arguments for the LSP server",
					"items": map[string]any{
						"type": "string",
					},
				},
				"options": map[string]any{
					"type":        "object",
					"description": "Additional options for the LSP server",
				},
			},
			"required": []string{"command"},
		},
	}

	return schema
}
