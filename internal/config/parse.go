package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// Raw AST structs for HCL decoding before validation.
type rawManifest struct {
	Server      *rawServer      `hcl:"server,block"`
	OpenAPI     *OpenAPI        `hcl:"openapi,block"`
	Problem     *Problem        `hcl:"problem,block"`
	Connections []rawConnection `hcl:"connection,block"`
	Schemas     []Schema        `hcl:"schema,block"`
	Endpoints   []rawEndpoint   `hcl:"endpoint,block"`
	Remain      hcl.Body        `hcl:",remain"`
}

type rawServer struct {
	Host         string   `hcl:"host,optional"`
	Port         int      `hcl:"port,optional"`
	ReadTimeout  string   `hcl:"read_timeout,optional"`
	WriteTimeout string   `hcl:"write_timeout,optional"`
	IdleTimeout  string   `hcl:"idle_timeout,optional"`
	MaxBodySize  string   `hcl:"max_body_size,optional"`
	Remain       hcl.Body `hcl:",remain"`
}

type rawConnection struct {
	Driver string      `hcl:"driver,label"`
	Name   string      `hcl:"name,label"`
	Source string      `hcl:"source,attr"`
	Pool   rawPoolTune `hcl:"pool,block"`
}

type rawPoolTune struct {
	MaxOpen     int    `hcl:"max_open,optional"`
	MaxIdle     int    `hcl:"max_idle,optional"`
	MaxLifetime string `hcl:"max_lifetime,optional"`
	IdleTimeout string `hcl:"idle_timeout,optional"`
}

type rawEndpoint struct {
	MethodAndPath string       `hcl:"name,label"`
	Description   *string      `hcl:"description,optional"`
	Request       *rawRequest  `hcl:"request,block"`
	Pipeline      *rawPipeline `hcl:"pipeline,block"`
	OpenAPIs      []rawOpenAPI `hcl:"openapi,block"`
	Remain        hcl.Body     `hcl:",remain"`
	DeclaringDir  string
}

type rawOpenAPI struct {
	Mode     string   `hcl:"mode,label"`
	Renderer *string  `hcl:"renderer,optional"`
	Format   *string  `hcl:"format,optional"`
	SpecURL  *string  `hcl:"spec_url,optional"`
	File     *string  `hcl:"file,optional"`
	Inline   *string  `hcl:"inline,optional"`
	Remain   hcl.Body `hcl:",remain"`
}

type rawPipeline struct {
	Body hcl.Body `hcl:",remain"`
}

type rawRequest struct {
	PathInline    *rawFieldGroup
	PathExpr      hcl.Expression
	QueryInline   *rawFieldGroup
	QueryExpr     hcl.Expression
	HeadersInline *rawFieldGroup
	HeadersExpr   hcl.Expression
	BodyInline    *rawFieldGroup
	BodyExpr      hcl.Expression
	Remain        hcl.Body `hcl:",remain"`
}

type rawFieldGroup struct {
	Fields []Field  `hcl:"field,block"`
	Remain hcl.Body `hcl:",remain"`
}

func (r *rawRequest) Decode(evalCtx *hcl.EvalContext) error {
	if r == nil || r.Remain == nil {
		return nil
	}

	content, _, diags := r.Remain.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "path", Required: false},
			{Name: "query", Required: false},
			{Name: "headers", Required: false},
			{Name: "body", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "path"},
			{Type: "query"},
			{Type: "headers"},
			{Type: "body"},
		},
	})
	if diags.HasErrors() {
		return fmt.Errorf("request block: %s", diags.Error())
	}

	if attr, ok := content.Attributes["path"]; ok {
		r.PathExpr = attr.Expr
	}
	if attr, ok := content.Attributes["query"]; ok {
		r.QueryExpr = attr.Expr
	}
	if attr, ok := content.Attributes["headers"]; ok {
		r.HeadersExpr = attr.Expr
	}
	if attr, ok := content.Attributes["body"]; ok {
		r.BodyExpr = attr.Expr
	}

	for _, block := range content.Blocks {
		var inline rawFieldGroup
		if err := gohcl.DecodeBody(block.Body, evalCtx, &inline); err.HasErrors() {
			return fmt.Errorf("inline %s block: %s", block.Type, err.Error())
		}
		switch block.Type {
		case "path":
			r.PathInline = &inline
		case "query":
			r.QueryInline = &inline
		case "headers":
			r.HeadersInline = &inline
		case "body":
			r.BodyInline = &inline
		}
	}
	return nil
}

// Load parses a single .hcl file or directory tree and returns a validated Config.
func Load(path string, evalCtx *hcl.EvalContext) (*Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("access path %q: %w", path, err)
	}

	p := hclparse.NewParser()
	var (
		merged      rawManifest
		serverFile  string
		openapiFile string
		problemFile string
	)

	walkFn := func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && currentPath != path {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.ToLower(filepath.Ext(d.Name())) == ".hcl" {
			file, diags := p.ParseHCLFile(currentPath)
			if diags.HasErrors() {
				return fmt.Errorf("parse %s: %s", currentPath, diags.Error())
			}

			var fileManifest rawManifest
			if diags := gohcl.DecodeBody(file.Body, evalCtx, &fileManifest); diags.HasErrors() {
				return fmt.Errorf("decode %s: %s", currentPath, diags.Error())
			}

			manifestDir := filepath.Dir(currentPath)
			if absDir, err := filepath.Abs(manifestDir); err == nil {
				manifestDir = absDir
			}

			// Validate singleton: server
			if fileManifest.Server != nil {
				if serverFile != "" {
					return fmt.Errorf("duplicate singleton block 'server' declared in %s and %s: only one 'server' block is permitted across all manifests", serverFile, currentPath)
				}
				merged.Server = fileManifest.Server
				serverFile = currentPath
			}

			// Validate singleton: openapi
			if fileManifest.OpenAPI != nil {
				if openapiFile != "" {
					return fmt.Errorf("duplicate singleton block 'openapi' declared in %s and %s: only one 'openapi' block is permitted across all manifests", openapiFile, currentPath)
				}
				merged.OpenAPI = fileManifest.OpenAPI
				openapiFile = currentPath
			}

			// Validate singleton: problem
			if fileManifest.Problem != nil {
				if problemFile != "" {
					return fmt.Errorf("duplicate singleton block 'problem' declared in %s and %s: only one 'problem' block is permitted across all manifests", problemFile, currentPath)
				}
				merged.Problem = fileManifest.Problem
				problemFile = currentPath
			}

			for i := range fileManifest.Endpoints {
				fileManifest.Endpoints[i].DeclaringDir = manifestDir
				if fileManifest.Endpoints[i].Request != nil {
					if err := fileManifest.Endpoints[i].Request.Decode(evalCtx); err != nil {
						return fmt.Errorf("endpoint %q: %w", fileManifest.Endpoints[i].MethodAndPath, err)
					}
				}
			}

			for i := range fileManifest.Connections {
				fileManifest.Connections[i].Source = resolveRelativePath(fileManifest.Connections[i].Source, manifestDir)
			}

			merged.Endpoints = append(merged.Endpoints, fileManifest.Endpoints...)
			merged.Connections = append(merged.Connections, fileManifest.Connections...)
			merged.Schemas = append(merged.Schemas, fileManifest.Schemas...)
		}
		return nil
	}

	if info.IsDir() {
		if err := filepath.WalkDir(path, walkFn); err != nil {
			return nil, err
		}
	} else {
		if err := walkFn(path, fs.FileInfoToDirEntry(info), nil); err != nil {
			return nil, err
		}
	}

	return validate(merged, evalCtx)
}

func resolveRelativePath(raw, baseDir string) string {
	if baseDir == "" || filepath.IsAbs(raw) || raw == ":memory:" || strings.Contains(raw, "://") {
		return raw
	}
	if strings.HasPrefix(raw, "file:") {
		rest := strings.TrimPrefix(raw, "file:")
		pathPart := rest
		queryPart := ""
		if idx := strings.Index(rest, "?"); idx != -1 {
			pathPart = rest[:idx]
			queryPart = rest[idx:]
		}
		if pathPart == ":memory:" || strings.Contains(pathPart, ":memory:") {
			return raw
		}
		absPath := pathPart
		if !filepath.IsAbs(pathPart) {
			absPath = filepath.Join(baseDir, pathPart)
		}
		return "file:" + filepath.ToSlash(absPath) + queryPart
	}
	return filepath.Join(baseDir, raw)
}
