package openapi

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// TemplateData holds template context data for OpenAPI HTML documentation viewers.
type TemplateData struct {
	Title       string
	Version     string
	Description string
	SpecURL     string
	SpecYAMLURL string
}

// RenderHTML generates the HTML page for a documentation UI renderer or custom template.
func RenderHTML(renderer, customTemplate string, data TemplateData) ([]byte, error) {
	var tmplContent string

	if customTemplate != "" {
		tmplContent = customTemplate
	} else {
		r := strings.ToLower(strings.TrimSpace(renderer))
		if r == "" {
			r = "scalar"
		}
		embeddedPath := fmt.Sprintf("templates/%s.html", r)
		content, err := templateFS.ReadFile(embeddedPath)
		if err != nil {
			return nil, fmt.Errorf("unsupported or missing documentation renderer %q", renderer)
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
