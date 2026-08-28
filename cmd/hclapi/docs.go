//go:build docs

package main

import (
	"fmt"
	"os"

	docs "github.com/urfave/cli-docs/v3"
)

func main() {
	cmd := newRootCommand()

	md, err := docs.ToTabularMarkdown(cmd, "hclapi")
	if err != nil {
		panic(err)
	}

	out := "# CLI Reference\n\n" +
		"<!-- This file is auto-generated. Do not edit by hand. -->\n\n" +
		md + "\n"

	if err := os.WriteFile("./docs/src/cli.md", []byte(out), 0o644); err != nil {
		panic(err)
	}

	fmt.Println("wrote docs/src/cli.md")
}
