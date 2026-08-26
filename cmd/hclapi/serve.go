package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ju4n97/hclapi"
	"github.com/urfave/cli/v3"
)

func serveCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Starts the Hclapi HTTP server.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to directory containing Hclapi manifests.",
				Value:   ".",
				Sources: cli.EnvVars("HCLAPI_CONFIG"),
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose debug logging.",
				Value:   false,
				Sources: cli.EnvVars("HCLAPI_VERBOSE"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			flagConfig := cmd.String("config")
			flagVerbose := cmd.Bool("verbose")

			logLevel := slog.LevelInfo
			if flagVerbose {
				logLevel = slog.LevelDebug
			}

			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: logLevel,
			}))

			logger.Info("booting Hclapi API engine...")

			engine, err := hclapi.NewEngine(hclapi.Options{
				ManifestDir:  flagConfig,
				StrictTyping: true,
				Logger:       logger,
			})
			if err != nil {
				logger.Error("engine initialization failed", "error", err)
				return err
			}

			server := &http.Server{
				Addr:         ":8080",
				Handler:      engine.Handler(),
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  60 * time.Second,
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				logger.Info("server started", "url", "http://localhost:8080")
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("server crashed", "error", err)
					os.Exit(1)
				}
			}()

			<-stop
			logger.Info("shutting down server...")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				logger.Error("forced shutdown error", "error", err)
				return err
			}

			slog.Info("server gracefully stopped")
			return nil
		},
	}
}
