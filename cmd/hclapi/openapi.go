package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ju4n97/hclapi/internal/compiler"
	"github.com/ju4n97/hclapi/internal/eval"
	"github.com/ju4n97/hclapi/internal/openapi"
	"github.com/ju4n97/hclapi/internal/parser"
)

func newOpenAPICommand() *cli.Command {
	return &cli.Command{
		Name:  "openapi",
		Usage: "Export the compiled OpenAPI 3.1 specification for your manifests.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c", "manifests", "m"},
				Usage:   "Path to .hcl file or directory containing manifests.",
				Value:   ".",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Path to output file (defaults to stdout).",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format: json or yaml.",
				Value:   "json",
			},
			&cli.BoolFlag{
				Name:  "pretty",
				Usage: "Pretty-print JSON output.",
				Value: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			evalCtx := eval.BaseContext()
			manifest, err := parser.Parse(cmd.String("config"), evalCtx)
			if err != nil {
				return fmt.Errorf("parse manifests: %w", err)
			}

			service, err := compiler.Compile(manifest, evalCtx)
			if err != nil {
				return fmt.Errorf("compile: %w", err)
			}

			var outBytes []byte
			if strings.EqualFold(cmd.String("format"), "yaml") || strings.EqualFold(cmd.String("format"), "yml") {
				outBytes, err = openapi.GenerateYAML(service)
			} else {
				outBytes, err = openapi.GenerateJSON(service, cmd.Bool("pretty"))
			}
			if err != nil {
				return fmt.Errorf("generate openapi: %w", err)
			}

			outputPath := cmd.String("output")
			if outputPath == "" {
				_, err = os.Stdout.Write(outBytes)
				return err
			}

			if err := os.WriteFile(outputPath, outBytes, 0o600); err != nil {
				return fmt.Errorf("write output file: %w", err)
			}
			return nil
		},
	}
}
