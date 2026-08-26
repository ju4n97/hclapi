package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	ctx := context.Background()

	cmd := &cli.Command{
		Name:  "hclapi",
		Usage: "Declarative API runtime that turns Hclapi manifests into structured HTTP services.",
		Commands: []*cli.Command{
			serveCommand(),
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.Error("cli execution failed", "error", err)
		os.Exit(1)
	}
}
