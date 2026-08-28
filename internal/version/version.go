package version

import "fmt"

var (
	version     = "dev"     // Git tag (e.g., "v1.2.3")
	branch      = "unknown" // Git branch (e.g., "main")
	shortCommit = "none"    // Git short commit SHA (e.g., "0fd6153")
	fullCommit  = "none"    // Git full commit SHA (e.g., "0fd6153429327455ec5bca2cda839116cfcb6a19")
	date        = "unknown" // Build date in RFC3339 format (e.g., "2025-03-02T18:46:18Z")
	os          = "unknown" // Operating system (e.g., "linux")
	arch        = "unknown" // CPU architecture (e.g., "amd64")
)

// String returns a formatted multi-line string that displays the build metadata.
func String() string {
	return fmt.Sprintf(
		"Version: %s\nBranch: %s\nShort commit: %s\nFull commit: %s\nBuild date: %s\nOS: %s\nArch: %s",
		version,
		branch,
		shortCommit,
		fullCommit,
		date,
		os,
		arch,
	)
}

// GetVersion returns the build version of the application.
func GetVersion() string {
	return version
}

// IsProd returns true if the application is running in production mode.
func IsProd() bool {
	return version != "" && version != "dev" && version != "test" && version != "local"
}

// GetBranch returns the Git branch name used during the build.
func GetBranch() string {
	return branch
}

// GetShortCommit returns the abbreviated Git Commit SHA from which the binary was built.
func GetShortCommit() string {
	return shortCommit
}

// GetFullCommit returns the full Git commit SHA from which the binary was built.
func GetFullCommit() string {
	return fullCommit
}

// GetDate returns the build date in RFC3339 format.
func GetDate() string {
	return date
}

// GetOS returns the operating system of target binary recorded at build time.
func GetOS() string {
	return os
}

// GetArch returns the CPU architecture of target binary recorded at build time.
func GetArch() string {
	return arch
}
