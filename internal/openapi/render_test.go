package openapi_test

import (
	"strings"
	"testing"

	"github.com/ju4n97/hclapi/internal/openapi"
)

func TestRenderHTML(t *testing.T) {
	t.Parallel()

	data := openapi.TemplateData{
		Title:       "Acme Storefront API",
		Version:     "1.2.0",
		Description: "Production API service",
		SpecURL:     "/openapi.json",
		SpecYAMLURL: "/openapi.yaml",
	}

	t.Run("Renders built-in Scalar UI", func(t *testing.T) {
		t.Parallel()

		htmlBytes, err := openapi.RenderHTML("scalar", "", data)
		if err != nil {
			t.Fatalf("failed to render scalar: %v", err)
		}

		htmlStr := string(htmlBytes)
		if !strings.Contains(htmlStr, "@scalar/api-reference") {
			t.Errorf("expected Scalar CDN in HTML")
		}
		if !strings.Contains(htmlStr, `data-url="/openapi.json"`) {
			t.Errorf("expected SpecURL in HTML")
		}
		if !strings.Contains(htmlStr, "Acme Storefront API (v1.2.0)") {
			t.Errorf("expected Title and Version in HTML")
		}
	})

	t.Run("Renders built-in Stoplight Elements UI", func(t *testing.T) {
		t.Parallel()

		htmlBytes, err := openapi.RenderHTML("elements", "", data)
		if err != nil {
			t.Fatalf("failed to render elements: %v", err)
		}

		htmlStr := string(htmlBytes)
		if !strings.Contains(htmlStr, "@stoplight/elements") {
			t.Errorf("expected Stoplight CDN in HTML")
		}
		if !strings.Contains(htmlStr, `apiDescriptionUrl="/openapi.json"`) {
			t.Errorf("expected SpecURL in Elements HTML")
		}
	})

	t.Run("Renders built-in Swagger UI", func(t *testing.T) {
		t.Parallel()

		htmlBytes, err := openapi.RenderHTML("swagger", "", data)
		if err != nil {
			t.Fatalf("failed to render swagger: %v", err)
		}

		htmlStr := string(htmlBytes)
		if !strings.Contains(htmlStr, "swagger-ui-bundle.js") {
			t.Errorf("expected Swagger UI bundle in HTML")
		}
		if !strings.Contains(htmlStr, `url: "/openapi.json"`) {
			t.Errorf("expected SpecURL in Swagger script")
		}
	})

	t.Run("Renders built-in Redoc UI", func(t *testing.T) {
		t.Parallel()

		htmlBytes, err := openapi.RenderHTML("redoc", "", data)
		if err != nil {
			t.Fatalf("failed to render redoc: %v", err)
		}

		htmlStr := string(htmlBytes)
		if !strings.Contains(htmlStr, "redoc.standalone.js") {
			t.Errorf("expected Redoc bundle in HTML")
		}
		if !strings.Contains(htmlStr, `spec-url="/openapi.json"`) {
			t.Errorf("expected SpecURL in Redoc element")
		}
	})

	t.Run("Renders custom inline HTML template string", func(t *testing.T) {
		t.Parallel()

		customTmpl := `<html><head><title>{{ .Title }}</title></head><body>Spec at {{ .SpecURL }}</body></html>`
		htmlBytes, err := openapi.RenderHTML("", customTmpl, data)
		if err != nil {
			t.Fatalf("failed to render custom template: %v", err)
		}

		htmlStr := string(htmlBytes)
		expected := "<html><head><title>Acme Storefront API</title></head><body>Spec at /openapi.json</body></html>"
		if htmlStr != expected {
			t.Errorf("got %q; want %q", htmlStr, expected)
		}
	})

	t.Run("Fails cleanly on unsupported renderer name", func(t *testing.T) {
		t.Parallel()

		_, err := openapi.RenderHTML("non_existent_renderer", "", data)
		if err == nil {
			t.Fatal("expected error on invalid renderer, got nil")
		}
	})
}
