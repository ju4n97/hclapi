package main

import (
	"github.com/urfave/cli/v3"

	"github.com/ju4n97/hclapi/internal/version"
)

func newRootCommand() *cli.Command {
	return &cli.Command{
		Name:                  "hclapi",
		Usage:                 "Declarative API runtime that turns HCL manifests into structured HTTP services.",
		Version:               version.GetVersion(),
		Suggest:               true,
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			newServeCommand(),
			newVersionCommand(),
		},
	}
}
