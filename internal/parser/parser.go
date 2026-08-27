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
)

func isHclapiManifest(filename string) bool {
	lower := strings.ToLower(filename)
	if lower == "hclapifile" {
		return true
	}
	ext := filepath.Ext(lower)
	return ext == ".hcl" || ext == ".hclapi"
}

// Parse reads a file or directory tree and returns the merged Manifest AST.
func Parse(path string) (*Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path %q: %w", path, err)
	}

	p := hclparse.NewParser()

	if !info.IsDir() {
		return parseFile(path, p)
	}

	var mergedManifest Manifest
	err = filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && currentPath != path {
			return filepath.SkipDir
		}

		if !d.IsDir() && isHclapiManifest(d.Name()) {
			fileManifest, err := parseFile(currentPath, p)
			if err != nil {
				return err
			}

			mergedManifest.Endpoints = append(mergedManifest.Endpoints, fileManifest.Endpoints...)
			if fileManifest.Server != nil {
				mergedManifest.Server = fileManifest.Server
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &mergedManifest, nil
}

func parseFile(path string, p *hclparse.Parser) (*Manifest, error) {
	file, diags := p.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file %s: %s", path, diags.Error())
	}

	var manifest Manifest
	diags = gohcl.DecodeBody(file.Body, nil, &manifest)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL file %s: %s", path, diags.Error())
	}

	return &manifest, nil
}

// DecodePipelineSteps decodes steps from a pipeline block preserving declaration order.
func DecodePipelineSteps(pipeline *PipelineBlock) ([]ParsedStep, error) {
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: string(StepTypeGo), LabelNames: []string{"name"}},
			{Type: string(StepTypeStarlark), LabelNames: []string{"name"}},
			{Type: string(StepTypeRespond)},
		},
	}

	content, diags := pipeline.Body.Content(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode pipeline steps: %s", diags.Error())
	}

	var steps []ParsedStep
	for _, block := range content.Blocks {
		switch block.Type {
		case string(StepTypeGo):
			var cfg GoStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("failed to decode go step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{
				Type: StepTypeGo,
				Name: block.Labels[0],
				Go:   &cfg,
			})

		case string(StepTypeStarlark):
			var cfg StarlarkStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("failed to decode starlark step %q: %s", block.Labels[0], diags.Error())
			}
			steps = append(steps, ParsedStep{
				Type:     StepTypeStarlark,
				Name:     block.Labels[0],
				Starlark: &cfg,
			})

		case string(StepTypeRespond):
			var cfg RespondStepBlock
			if diags := gohcl.DecodeBody(block.Body, nil, &cfg); diags.HasErrors() {
				return nil, fmt.Errorf("failed to decode respond step: %s", diags.Error())
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
