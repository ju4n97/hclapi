package hclapi

// Options defines the configuration for the Hclapi engine.
type Options struct {
	// ManifestDir is the path to the directory containing .hcl files.
	ManifestDir string

	// StrictTyping ensures all schema references are strictly validated.
	StrictTyping bool
}
