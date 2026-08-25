package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ekisa-team/sqlmux/config"
	"github.com/fsnotify/fsnotify"
)

const (
	defaultConfigPath = "./sqlmux.yaml"
	reloadDebounce    = 100 * time.Millisecond
)

func main() {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)
	slog.SetDefault(logger)

	configPath := os.Getenv("SQLMUX_CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	configPath, err := filepath.Abs(configPath)
	if err != nil {
		slog.Error("Failed to resolve absolute path", "err", err, "path", configPath)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if _, err := reload(configPath); err != nil {
		slog.Error("Failed to load initial config", "err", err, "path", configPath)
		os.Exit(1)
	}

	if err := watchConfig(ctx, configPath); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("config watcher stopped", "err", err)
		os.Exit(1)
	}

	slog.Info("shutting down")
}

func watchConfig(ctx context.Context, configPath string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	configDir := filepath.Dir(configPath)
	configName := filepath.Base(configPath)

	if err := watcher.Add(configDir); err != nil {
		return err
	}

	slog.Info(
		"watching config",
		"path", configPath,
		"directory", configDir,
	)

	var debounceTimer *time.Timer
	var debounce <-chan time.Time

	triggerReload := func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}

		debounceTimer = time.NewTimer(reloadDebounce)
		debounce = debounceTimer.C
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-debounce:
			debounce = nil

			if _, err := reload(configPath); err != nil {
				slog.Error("Failed to reload config", "err", err, "path", configPath)
			}

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if filepath.Base(event.Name) != configName {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			slog.Debug("config filesystem event", "path", event.Name, "op", event.Op.String())

			triggerReload()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			slog.Error("config filesystem error", "err", err)
		}
	}
}

func reload(configPath string) (*config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	slog.Info(
		"config loaded",
		"path", configPath,
	)

	return cfg, nil
}
