package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ju4n97/hclapi/internal/version"
)

func newVersionCommand() *cli.Command {
	return &cli.Command{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Show detailed version information.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(version.GetVersion())
			return nil
		},
	}
}
