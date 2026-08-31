//go:build docs

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"
)

const outputDir = "./docs/content/cli"

func main() {
	if err := generateDocs(newRootCommand()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// generateDocs recreates outputDir and writes one Markdown file per command.
// It tracks command paths explicitly because cli.Command's path helpers
// depend on parent pointers that are only wired during Run.
func generateDocs(root *cli.Command) error {
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("remove generated docs: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	count, err := writeCommandDocs(root, nil)
	if err != nil {
		return err
	}

	fmt.Printf("wrote %d CLI docs to %s\n", count, outputDir)
	return nil
}

// writeCommandDocs writes docs for cmd and its declared subcommands.
// path contains the command names leading to cmd.
func writeCommandDocs(cmd *cli.Command, path []string) (int, error) {
	if cmd.Hidden {
		return 0, nil
	}

	path = append(append([]string(nil), path...), cmd.Name)

	if err := writeCommandDoc(cmd, path); err != nil {
		return 0, err
	}

	count := 1
	for _, sub := range cmd.Commands {
		n, err := writeCommandDocs(sub, path)
		if err != nil {
			return 0, err
		}
		count += n
	}

	return count, nil
}

// writeCommandDoc generates and writes the Markdown documentation for cmd.
func writeCommandDoc(cmd *cli.Command, path []string) error {
	name := strings.Join(path, " ")

	md, err := docs.ToMarkdown(cmd)
	if err != nil {
		return fmt.Errorf("generate %q: %w", name, err)
	}

	content := fmt.Sprintf(
		"---\ntitle: %s\n---\n\n"+
			"<!-- This file is auto-generated. Do not edit by hand. -->\n\n"+
			"%s\n",
		name,
		demoteHeadings(md),
	)

	filePath := filepath.Join(outputDir, commandFilename(path))
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("wrote %s\n", filePath)
	return nil
}

// commandFilename returns the output filename for a command path.
func commandFilename(path []string) string {
	rest := path[1:]
	if len(rest) == 0 {
		return "hclapi.md"
	}
	return "hclapi-" + strings.Join(rest, "-") + ".md"
}

// demoteHeadings nests generated headings under the page title.
func demoteHeadings(md string) string {
	lines := strings.Split(md, "\n")
	first := true

	for i, line := range lines {
		if !strings.HasPrefix(line, "#") {
			continue
		}

		if first {
			lines[i] = "#" + line
			first = false
		} else {
			lines[i] = "##" + line
		}
	}

	return strings.Join(lines, "\n")
}
