package parser

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// ParseDir reads all *.hcl files in a directory and merges them into a single Manifest.
func ParseDir(dir string) (*Manifest, error) {
	matches, err := filepath.Glob(path.Join(dir, "*.hcl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list HCL files: %w", err)
	}

	p := hclparse.NewParser()
	var mergedManifest Manifest

	for _, match := range matches {
		file, diags := p.ParseHCLFile(match)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse HCL file: %w", diags)
		}

		var fileManifest Manifest
		diags = gohcl.DecodeBody(file.Body, nil, &fileManifest)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to decode HCL file: %w", diags)
		}

		mergedManifest.Endpoints = append(mergedManifest.Endpoints, fileManifest.Endpoints...)
	}

	return &mergedManifest, nil
}

// ParseBytes parses a raw HCL bytes slice. This is primarily used for testing.
func ParseBytes(src []byte, filename string) (*Manifest, error) {
	p := hclparse.NewParser()
	file, diags := p.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file: %w", diags)
	}

	var manifest Manifest
	diags = gohcl.DecodeBody(file.Body, nil, &manifest)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode HCL file: %w", diags)
	}

	return &manifest, nil
}
