package parser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// ParseDir reads all *.hcl files in a directory and merges them into a single Manifest.
func ParseDir(dir string) (*Manifest, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	matches, err := filepath.Glob(path.Join(dir, "*.hcl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list HCL files: %w", err)
	}

	p := hclparse.NewParser()
	var mergedManifest Manifest

	for _, match := range matches {
		file, diags := p.ParseHCLFile(match)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse HCL file %s: %s", match, diags.Error())
		}

		var fileManifest Manifest
		diags = gohcl.DecodeBody(file.Body, nil, &fileManifest)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to decode HCL file %s: %s", match, diags.Error())
		}

		mergedManifest.Endpoints = append(mergedManifest.Endpoints, fileManifest.Endpoints...)
	}

	return &mergedManifest, nil
}
