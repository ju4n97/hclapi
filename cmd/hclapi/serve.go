package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/ju4n97/hclapi"
)

func newServeCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Starts the hclapi HTTP server.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c", "manifests", "m"},
				Usage:   "Path to .hcl file, or directory containing manifests.",
				Value:   ".",
				Sources: cli.EnvVars("HCLAPI_CONFIG", "HCLAPI_MANIFESTS"),
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
			&cli.StringFlag{
				Name:    "host",
				Aliases: []string{"h"},
				Usage:   "Host address to bind the server (overrides manifest).",
				Sources: cli.EnvVars("HCLAPI_HOST", "HOST"),
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Port to bind the server (overrides manifest).",
				Sources: cli.EnvVars("HCLAPI_PORT", "PORT"),
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
			logLevel := slog.LevelInfo
			if cmd.Bool("verbose") {
				logLevel = slog.LevelDebug
			}

			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: logLevel,
			}))

			logger.Info("booting hclapi API engine...")

			engine, err := hclapi.NewEngine(hclapi.Options{
				ConfigPath:   cmd.String("config"),
				StrictTyping: true,
				Logger:       logger,
			})
			if err != nil {
				logger.Error("engine initialization failed", "error", err)
				return fmt.Errorf("engine initialization failed: %w", err)
			}

			srv := engine.Server()
			host := srv.Host
			port := srv.Port
			if cmd.IsSet("host") {
				host = cmd.String("host")
			}
			if cmd.IsSet("port") {
				port = cmd.Int("port")
			}

			server := &http.Server{
				Addr:         fmt.Sprintf("%s:%d", host, port),
				Handler:      engine.Handler(),
				ReadTimeout:  srv.ReadTimeout.Duration(),
				WriteTimeout: srv.WriteTimeout.Duration(),
				IdleTimeout:  srv.IdleTimeout.Duration(),
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

			go func() {
				logger.Info("server started", "addr", server.Addr)
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("server crashed", "error", err)
					os.Exit(1)
				}
			}()

			<-stop
			logger.Info("shutting down server...")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			return server.Shutdown(shutdownCtx)
		},
	}
}
