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
		Usage: "Start the hclapi HTTP server.",
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
				Config: cli.StringConfig{
					TrimSpace: true,
				},
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
				return err
			}

			srv := engine.Server()

			host := srv.Host
			if cmd.IsSet("host") {
				host = cmd.String("host")
			}

			port := srv.Port
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

			errCh := make(chan error, 1)

			go func() {
				logger.Info("server started", "addr", server.Addr)

				if err := server.ListenAndServe(); err != nil &&
					!errors.Is(err, http.ErrServerClosed) {
					errCh <- err
				}
			}()

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(stop)

			select {
			case err := <-errCh:
				logger.Error("server crashed", "error", err)
				return err
			case <-stop:
				logger.Info("shutting down server...")
			case <-ctx.Done():
				logger.Info("context cancelled, shutting down server...")
			}

			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("shutdown server: %w", err)
			}

			return nil
		},
	}
}
