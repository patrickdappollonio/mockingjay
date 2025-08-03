package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/patrickdappollonio/mockingjay/internal/config"
	"github.com/patrickdappollonio/mockingjay/internal/server"
)

var (
	// Version is set via goreleaser ldflags
	version = "dev"

	configFile   = flag.String("config", "config.yaml", "path to configuration file")
	port         = flag.String("port", "8080", "server port")
	debug        = flag.Bool("debug", false, "enable debug logging")
	validateOnly = flag.Bool("validate", false, "validate configuration file and exit")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Set up structured logging
	logger := setupLogger(*debug)

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Error("failed to load configuration", "file", *configFile, "error", err)
		return err
	}

	logger.Info("configuration loaded successfully",
		"file", *configFile,
		"routes_count", len(cfg.Routes),
	)

	// If validation-only mode, exit after successful validation
	if *validateOnly {
		logger.Info("configuration validation completed successfully")
		fmt.Printf("✅ Configuration file %q is valid\n", *configFile)
		fmt.Printf("   - Found %d routes\n", len(cfg.Routes))
		fmt.Printf("   - All validation checks passed\n")
		return nil
	}

	// Create server
	addr := ":" + *port
	srv, err := server.NewServer(cfg, *configFile, addr, logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		return err
	}

	// Create context that cancels on interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start config file watcher for hot-reload
	if err := startConfigWatcher(*configFile, srv, logger, ctx); err != nil {
		logger.Error("failed to start config file watcher", "error", err)
		return err
	}

	// Start server
	logger.Info("starting mockingjay server", "version", version, "addr", addr)
	if err := srv.Start(ctx); err != nil {
		logger.Error("server error", "error", err)
		return err
	}

	logger.Info("server stopped gracefully")
	return nil
}

// setupLogger configures structured logging based on debug mode
func setupLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: debug, // Add source file info in debug mode
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

// startConfigWatcher starts a file watcher to monitor config changes for hot-reload
func startConfigWatcher(configFile string, srv *server.Server, logger *slog.Logger, ctx context.Context) error {
	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Add config file to watcher
	if err := watcher.Add(configFile); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch config file %q: %w", configFile, err)
	}

	logger.Info("config file watcher started", "file", configFile)

	// Start watcher in background goroutine
	go func() {
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				logger.Debug("config watcher stopping due to context cancellation")
				return

			case event, ok := <-watcher.Events:
				if !ok {
					logger.Debug("config watcher events channel closed")
					return
				}

				// Only handle write events (file modifications)
				if event.Op&fsnotify.Write == fsnotify.Write {
					logger.Info("config file changed, reloading", "file", event.Name)

					if err := srv.ReloadConfig(); err != nil {
						logger.Error("failed to reload config", "error", err)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					logger.Debug("config watcher errors channel closed")
					return
				}
				logger.Error("config file watcher error", "error", err)
			}
		}
	}()

	return nil
}
