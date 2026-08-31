package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"

	"github.com/ju4n97/hclapi/internal/core"
)

func ishclapiManifest(filename string) bool {
	lower := strings.ToLower(filename)
	ext := filepath.Ext(lower)
	return ext == ".hcl"
}

// Parse reads a file or directory tree and returns the merged Manifest AST.
func Parse(path string, evalCtx *hcl.EvalContext) (*Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("access path %q: %w", path, err)
	}

	p := hclparse.NewParser()

	if !info.IsDir() {
		return parseFile(path, p, evalCtx)
	}

	var mergedManifest Manifest
	err = filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && currentPath != path {
			return filepath.SkipDir
		}

		if !d.IsDir() && ishclapiManifest(d.Name()) {
			fileManifest, err := parseFile(currentPath, p, evalCtx)
			if err != nil {
				return err
			}

			mergedManifest.Endpoints = append(mergedManifest.Endpoints, fileManifest.Endpoints...)
			mergedManifest.Connections = append(mergedManifest.Connections, fileManifest.Connections...)
			if fileManifest.Server != nil {
				mergedManifest.Server = fileManifest.Server
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Validate unique connection keys (<driver>.<name>)
	seenConnections := make(map[string]bool, len(mergedManifest.Connections))
	for _, conn := range mergedManifest.Connections {
		key := fmt.Sprintf("%s.%s", conn.Driver, conn.Name)
		if seenConnections[key] {
			return nil, fmt.Errorf("duplicate connection declaration %q", "connection."+key)
		}
		seenConnections[key] = true
	}

	return &mergedManifest, nil
}

func parseFile(path string, p *hclparse.Parser, evalCtx *hcl.EvalContext) (*Manifest, error) {
	file, diags := p.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse %s: %s", path, diags.Error())
	}

	var manifest Manifest
	diags = gohcl.DecodeBody(file.Body, evalCtx, &manifest)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decode %s: %s", path, diags.Error())
	}

	manifestDir, err := filepath.Abs(filepath.Dir(path))
	if err == nil {
		for i := range manifest.Connections {
			manifest.Connections[i].URL = core.ResolveRelativePath(manifest.Connections[i].URL, manifestDir)
		}
	}

	return &manifest, nil
}

// DecodePipelineSteps decodes steps from a pipeline block preserving declaration order.
func DecodePipelineSteps(pipeline *PipelineBlock) ([]ParsedStep, error) {
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: string(StepTypeGo), LabelNames: []string{"name"}},
			{Type: string(StepTypeStarlark), LabelNames: []string{"name"}},
			{Type: string(StepTypeSQL), LabelNames: []string{"name"}},
			{Type: string(StepTypeRespond)},
		},
	}

	content, diags := pipeline.Body.Content(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decode pipeline: %s", diags.Error())
	}

	var steps []ParsedStep
	for _, block := range content.Blocks {
		switch block.Type {
		case string(StepTypeGo):
			var cfg GoStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("go step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{
				Type: StepTypeGo,
				Name: block.Labels[0],
				Go:   &cfg,
			})

		case string(StepTypeStarlark):
			var cfg StarlarkStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("starlark step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{
				Type:     StepTypeStarlark,
				Name:     block.Labels[0],
				Starlark: &cfg,
			})

		case string(StepTypeSQL):
			var cfg SQLStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("sql step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{
				Type: StepTypeSQL,
				Name: block.Labels[0],
				SQL:  &cfg,
			})

		case string(StepTypeRespond):
			var cfg RespondStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("respond step: %s", diags.Error())
			}
			steps = append(steps, ParsedStep{
				Type:    StepTypeRespond,
				Respond: &cfg,
			})

		default:
			return nil, fmt.Errorf("unknown step type %q", block.Type)
		}
	}

	return steps, nil
}
