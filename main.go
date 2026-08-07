package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	fyneapp "fyne.io/fyne/v2/app"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/db"
	"github.com/cindyhuang123/hylbscode/internal/gui"
	"github.com/cindyhuang123/hylbscode/internal/logging"
)

func fatal(format string, args ...any) {
	logging.ErrorPersist(format, args...)
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	debug := flag.Bool("d", false, "enable debug logging")
	flag.Parse()

	workingDir, err := os.Getwd()
	if err != nil {
		fatal("failed to get working directory: %v", err)
	}
	cfg, err := config.Load(workingDir, *debug)
	if err != nil {
		fatal("failed to load configuration: %v", err)
	}
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	// 日志同时写入 {data.directory}/hylbscode.log，方便离线排查 GUI 交互问题。
	if err := os.MkdirAll(cfg.Data.Directory, 0o755); err != nil {
		fatal("failed to create data directory for logs: %v", err)
	}
	logFile, err := os.OpenFile(filepath.Join(cfg.Data.Directory, "hylbscode.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		fatal("failed to open log file: %v", err)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, logFile), &slog.HandlerOptions{Level: level})))
	if !config.HasProviderCredentials() {
		fmt.Fprintln(os.Stderr, "warning: no LLM provider credentials detected in environment variables.")
		fmt.Fprintln(os.Stderr, "  (providers configured in the config file are still honored)")
	}

	conn, err := db.Connect()
	if err != nil {
		fatal("failed to connect to database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	core, err := app.New(ctx, conn)
	if err != nil {
		fatal("failed to initialize application: %v", err)
	}

	a := fyneapp.NewWithID("com.hylbscode.desktop")
	g := gui.NewMainWindow(a, core, ctx)
	cancel := gui.SetupSubscriptions(g, ctx)
	g.Window().SetOnClosed(cancel)
	g.Show()
	if !config.HasProviderCredentials() {
		g.ShowProviderSetup()
	}
	a.Run()
}
