package main

import (
	"log/slog"
	"os"

	"github.com/ekisa-team/sqlmux/config"
)

func main() {
	slog.SetDefault(
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	)

	configPath := os.Getenv("SQLMUX_CONFIG")
	if configPath == "" {
		configPath = "./sqlmux.yaml"
	}

	config, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	slog.Info("Loaded config", "config", configPath)

	slog.Info("Loaded config", "config", config)
}
