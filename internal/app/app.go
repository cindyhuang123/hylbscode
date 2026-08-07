package app

import (
	"context"
	"database/sql"
	"maps"
	"sync"
	"time"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/db"
	"github.com/cindyhuang123/hylbscode/internal/history"
	"github.com/cindyhuang123/hylbscode/internal/llm/agent"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/lsp"
	"github.com/cindyhuang123/hylbscode/internal/message"
	"github.com/cindyhuang123/hylbscode/internal/permission"
	"github.com/cindyhuang123/hylbscode/internal/search"
	"github.com/cindyhuang123/hylbscode/internal/session"
	"github.com/cindyhuang123/hylbscode/internal/todo"
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service
	Search      search.Service
	Todos       todo.Service

	CoderAgent agent.Service

	LSPClients map[string]*lsp.Client

	clientsMutex sync.RWMutex

	watcherCancelFuncs []context.CancelFunc
	cancelFuncsMutex   sync.Mutex
	watcherWG          sync.WaitGroup
}

func New(ctx context.Context, conn *sql.DB) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	files := history.NewService(q, conn)
	messageSearch := search.NewService(conn)
	todos := todo.NewService(q)

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(),
		Search:      messageSearch,
		Todos:       todos,
		LSPClients:  make(map[string]*lsp.Client),
	}

	// Initialize LSP clients in the background
	go app.initLSPClients(ctx)

	// Coder agent 创建失败时不致命：没配置 API Key 时也要能启动 GUI
	// 让用户通过设置界面配置，配置保存后再重建（见 EnsureCoderAgent）。
	if err := app.EnsureCoderAgent(); err != nil {
		logging.Warn("coder agent not created yet, will prompt for provider setup", "err", err)
	}

	return app, nil
}

// EnsureCoderAgent 创建 coder agent（若尚未创建）。在启动时或用户配置完
// API Key 后调用，返回错误时 CoderAgent 保持为空。
func (a *App) EnsureCoderAgent() error {
	if a.CoderAgent != nil {
		return nil
	}
	agentSvc, err := agent.NewAgent(
		config.AgentCoder,
		a.Sessions,
		a.Messages,
		agent.CoderAgentTools(
			a.Permissions,
			a.Sessions,
			a.Messages,
			a.History,
			a.Search,
			a.Todos,
			a.LSPClients,
		),
	)
	if err != nil {
		return err
	}
	a.CoderAgent = agentSvc
	return nil
}

// Shutdown performs a clean shutdown of the application
func (app *App) Shutdown() {
	agent.CloseMcpClients()

	// Cancel all watcher goroutines
	app.cancelFuncsMutex.Lock()
	for _, cancel := range app.watcherCancelFuncs {
		cancel()
	}
	app.cancelFuncsMutex.Unlock()
	app.watcherWG.Wait()

	// Perform additional cleanup for LSP clients
	app.clientsMutex.RLock()
	clients := make(map[string]*lsp.Client, len(app.LSPClients))
	maps.Copy(clients, app.LSPClients)
	app.clientsMutex.RUnlock()

	for name, client := range clients {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := client.Shutdown(shutdownCtx); err != nil {
			logging.Error("Failed to shutdown LSP client", "name", name, "error", err)
		}
		cancel()
	}
}
