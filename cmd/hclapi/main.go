package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	ctx := context.Background()

	if err := newRootCommand().Run(ctx, os.Args); err != nil {
		slog.Error("cli execution failed", "error", err)
		os.Exit(1)
	}
}
