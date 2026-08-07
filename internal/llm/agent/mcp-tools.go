package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/tools"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/permission"
	"github.com/cindyhuang123/hylbscode/internal/version"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpTool struct {
	mcpName     string
	tool        mcp.Tool
	mcpConfig   config.MCPServer
	permissions permission.Service
}

type MCPClient interface {
	Initialize(
		ctx context.Context,
		request mcp.InitializeRequest,
	) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Close() error
}

type managedClient struct {
	mu     sync.Mutex
	client MCPClient
	cfg    config.MCPServer
}

const (
	pingTimeout = 2 * time.Second
	callTimeout = 60 * time.Second
)

type pinger interface {
	Ping(context.Context) error
}

func (mc *managedClient) alive(ctx context.Context) bool {
	p, ok := mc.client.(pinger)
	if !ok {
		return true
	}
	pctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	return p.Ping(pctx) == nil
}

func (mc *managedClient) ensure(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.ensureLocked(ctx)
}

func (mc *managedClient) ensureLocked(ctx context.Context) error {
	if mc.client != nil {
		return nil
	}
	c, err := newMCPClient(mc.cfg)
	if err != nil {
		return err
	}
	if err := initialize(c, ctx); err != nil {
		c.Close()
		return err
	}
	mc.client = c
	return nil
}

func (mc *managedClient) callLocked(ctx context.Context, toolName, input string) (tools.ToolResponse, error) {
	toolRequest := mcp.CallToolRequest{}
	toolRequest.Params.Name = toolName
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	toolRequest.Params.Arguments = args
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	result, err := mc.client.CallTool(callCtx, toolRequest)
	if err != nil {
		return tools.ToolResponse{}, err
	}
	output := ""
	for _, v := range result.Content {
		if v, ok := v.(mcp.TextContent); ok {
			output = v.Text
		} else {
			output = fmt.Sprintf("%v", v)
		}
	}
	return tools.NewTextResponse(output), nil
}

type mcpClientManager struct {
	mu      sync.Mutex
	clients map[string]*managedClient
}

var mcpManager = &mcpClientManager{clients: make(map[string]*managedClient)}

func (m *mcpClientManager) get(name string, cfg config.MCPServer) *managedClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	mc, ok := m.clients[name]
	if !ok {
		mc = &managedClient{cfg: cfg}
		m.clients[name] = mc
	}
	return mc
}

func (m *mcpClientManager) remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mc, ok := m.clients[name]; ok {
		mc.mu.Lock()
		if mc.client != nil {
			mc.client.Close()
		}
		mc.mu.Unlock()
		delete(m.clients, name)
	}
}

func (m *mcpClientManager) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, mc := range m.clients {
		mc.mu.Lock()
		if mc.client != nil {
			mc.client.Close()
		}
		mc.mu.Unlock()
		delete(m.clients, name)
	}
}

func newMCPClient(cfg config.MCPServer) (MCPClient, error) {
	switch cfg.Type {
	case config.MCPStdio:
		return client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
	case config.MCPSse:
		return client.NewSSEMCPClient(cfg.URL, client.WithHeaders(cfg.Headers))
	}
	return nil, fmt.Errorf("invalid mcp server type: %s", cfg.Type)
}

func initialize(c MCPClient, ctx context.Context) error {
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "HyLbsCode",
		Version: version.Version,
	}
	_, err := c.Initialize(ctx, initRequest)
	return err
}

func (b *mcpTool) Info() tools.ToolInfo {
	required := b.tool.InputSchema.Required
	if required == nil {
		required = make([]string, 0)
	}
	return tools.ToolInfo{
		Name:        fmt.Sprintf("%s_%s", b.mcpName, b.tool.Name),
		Description: b.tool.Description,
		Parameters:  b.tool.InputSchema.Properties,
		Required:    required,
	}
}

func (b *mcpTool) Run(ctx context.Context, params tools.ToolCall) (tools.ToolResponse, error) {
	sessionID, messageID := tools.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return tools.ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	permissionDescription := fmt.Sprintf("execute %s with the following parameters: %s", b.Info().Name, params.Input)
	p := b.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        config.WorkingDirectory(),
			ToolName:    b.Info().Name,
			Action:      "execute",
			Description: permissionDescription,
			Params:      params.Input,
		},
	)
	if !p {
		return tools.NewTextErrorResponse("permission denied"), nil
	}

	mc := mcpManager.get(b.mcpName, b.mcpConfig)
	response, err := b.call(ctx, mc, params.Input)
	if err == nil {
		return response, nil
	}
	// Connection-level failure: drop the stale client and retry once with a fresh one.
	mcpManager.remove(b.mcpName)
	mc = mcpManager.get(b.mcpName, b.mcpConfig)
	response, err = b.call(ctx, mc, params.Input)
	if err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("MCP server %s unavailable: %s", b.mcpName, err)), nil
	}
	return response, nil
}

func (b *mcpTool) call(ctx context.Context, mc *managedClient, input string) (tools.ToolResponse, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if err := mc.ensureLocked(ctx); err != nil {
		return tools.ToolResponse{}, fmt.Errorf("failed to connect to MCP server %s: %w", b.mcpName, err)
	}
	if !mc.alive(ctx) {
		return tools.ToolResponse{}, fmt.Errorf("MCP server %s connection lost", b.mcpName)
	}
	return mc.callLocked(ctx, b.tool.Name, input)
}

func NewMcpTool(name string, tool mcp.Tool, permissions permission.Service, mcpConfig config.MCPServer) tools.BaseTool {
	return &mcpTool{
		mcpName:     name,
		tool:        tool,
		mcpConfig:   mcpConfig,
		permissions: permissions,
	}
}

var (
	mcpToolsMu     sync.Mutex
	mcpServerTools = map[string][]tools.BaseTool{}
)

func registerServerTools(ctx context.Context, name string, m config.MCPServer, permissions permission.Service) []tools.BaseTool {
	mc := mcpManager.get(name, m)
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if err := mc.ensureLocked(ctx); err != nil {
		logging.Error("error connecting to mcp server", "server", name, "error", err)
		return nil
	}
	toolsRequest := mcp.ListToolsRequest{}
	toolList, err := mc.client.ListTools(ctx, toolsRequest)
	if err != nil {
		logging.Error("error listing tools", "server", name, "error", err)
		return nil
	}
	registered := make([]tools.BaseTool, 0, len(toolList.Tools))
	for _, t := range toolList.Tools {
		registered = append(registered, NewMcpTool(name, t, permissions, m))
	}
	return registered
}

func GetMcpTools(ctx context.Context, permissions permission.Service) []tools.BaseTool {
	mcpToolsMu.Lock()
	defer mcpToolsMu.Unlock()
	for name, m := range config.Get().MCPServers {
		if _, ok := mcpServerTools[name]; ok {
			continue
		}
		if registered := registerServerTools(ctx, name, m, permissions); registered != nil {
			mcpServerTools[name] = registered
		}
	}
	var all []tools.BaseTool
	for _, ts := range mcpServerTools {
		all = append(all, ts...)
	}
	return all
}

func CloseMcpClients() {
	mcpManager.closeAll()
}
