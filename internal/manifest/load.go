package manifest

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

type fileManifest struct {
	Server      *ServerBlock      `hcl:"server,block"`
	OpenAPI     *OpenAPIBlock     `hcl:"openapi,block"`
	Problem     *ProblemBlock     `hcl:"problem,block"`
	Connections []ConnectionBlock `hcl:"connection,block"`
	Schemas     []SchemaBlock     `hcl:"schema,block"`
	Endpoints   []EndpointBlock   `hcl:"endpoint,block"`
	Remain      hcl.Body          `hcl:",remain"`
}

// Load parses all .hcl files in a target directory or file and returns an aggregate Manifest.
func Load(path string, evalCtx *hcl.EvalContext) (*Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("access path %q: %w", path, err)
	}

	p := hclparse.NewParser()
	var (
		merged      Manifest
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

			var fileCfg fileManifest
			if diags := gohcl.DecodeBody(file.Body, evalCtx, &fileCfg); diags.HasErrors() {
				return fmt.Errorf("decode %s: %s", currentPath, diags.Error())
			}

			manifestDir := filepath.Dir(currentPath)
			if absDir, err := filepath.Abs(manifestDir); err == nil {
				manifestDir = absDir
			}

			if fileCfg.Server != nil {
				if serverFile != "" {
					return fmt.Errorf(
						"duplicate singleton block 'server' declared in %s and %s: only one 'server' block is permitted across all manifests",
						serverFile,
						currentPath,
					)
				}
				merged.Server = fileCfg.Server
				serverFile = currentPath
			}

			if fileCfg.OpenAPI != nil {
				if openapiFile != "" {
					return fmt.Errorf(
						"duplicate singleton block 'openapi' declared in %s and %s: only one 'openapi' block is permitted across all manifests",
						openapiFile,
						currentPath,
					)
				}
				merged.OpenAPI = fileCfg.OpenAPI
				openapiFile = currentPath
			}

			if fileCfg.Problem != nil {
				if problemFile != "" {
					return fmt.Errorf(
						"duplicate singleton block 'problem' declared in %s and %s: only one 'problem' block is permitted across all manifests",
						problemFile,
						currentPath,
					)
				}
				merged.Problem = fileCfg.Problem
				problemFile = currentPath
			}

			for i := range fileCfg.Endpoints {
				fileCfg.Endpoints[i].DeclaringDir = manifestDir
				if fileCfg.Endpoints[i].Request != nil {
					if err := fileCfg.Endpoints[i].Request.Decode(evalCtx); err != nil {
						return fmt.Errorf("endpoint %q: %w", fileCfg.Endpoints[i].MethodAndPath, err)
					}
				}
			}

			for i := range fileCfg.Connections {
				fileCfg.Connections[i].DeclaringDir = manifestDir
				fileCfg.Connections[i].Source = resolveRelativePath(fileCfg.Connections[i].Source, manifestDir)
			}

			merged.Endpoints = append(merged.Endpoints, fileCfg.Endpoints...)
			merged.Connections = append(merged.Connections, fileCfg.Connections...)
			merged.Schemas = append(merged.Schemas, fileCfg.Schemas...)
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

	return &merged, nil
}

func resolveRelativePath(raw, baseDir string) string {
	if baseDir == "" || filepath.IsAbs(raw) || raw == ":memory:" || strings.Contains(raw, "://") {
		return raw
	}
	if after, ok := strings.CutPrefix(raw, "file:"); ok {
		rest := after
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
