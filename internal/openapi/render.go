package openapi

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/ju4n97/hclapi/internal/core"
)

//go:embed templates/*
var templateFS embed.FS

// TemplateData holds view context data for OpenAPI HTML renderers.
type TemplateData struct {
	Title       string
	Version     string
	Description string
	SpecURL     string
	SpecYAMLURL string
}

// RenderHTML generates the HTML page for the selected interactive documentation renderer.
func RenderHTML(ui string, data TemplateData, rawTemplate, templateFile, manifestDir string) ([]byte, error) {
	var tmplContent string

	switch {
	case rawTemplate != "":
		tmplContent = rawTemplate
	case templateFile != "":
		resolvedPath := core.ResolveRelativePath(templateFile, manifestDir)
		content, err := os.ReadFile(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("read custom template file: %w", err)
		}
		tmplContent = string(content)
	default:
		rendererName := strings.ToLower(strings.TrimSpace(ui))
		if rendererName == "" {
			rendererName = "scalar"
		}
		embeddedPath := fmt.Sprintf("templates/%s.html", rendererName)
		content, err := templateFS.ReadFile(embeddedPath)
		if err != nil {
			return nil, fmt.Errorf("unsupported or missing documentation renderer %q", ui)
		}
		tmplContent = string(content)
	}

	tmpl, err := template.New("docs").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parse docs template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute docs template: %w", err)
	}

	return buf.Bytes(), nil
}
