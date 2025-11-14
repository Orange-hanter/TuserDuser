package main

import (
	"fmt"
	"runtime"
)

var (
	// Version holds the semantic version injected at build time via -ldflags.
	Version = "dev"
	// Commit holds the short git commit SHA injected at build time via -ldflags.
	Commit = "none"
	// BuildTime holds the RFC3339 build timestamp injected at build time via -ldflags.
	BuildTime = "unknown"
)

// VersionInfo aggregates build metadata exposed via CLI, HTTP, and logs.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

func newVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   fallback(Version, "dev"),
		Commit:    fallback(Commit, "none"),
		BuildTime: fallback(BuildTime, "unknown"),
		GoVersion: runtime.Version(),
	}
}

// String formats VersionInfo as a single human-readable line.
func (vi VersionInfo) String() string {
	return fmt.Sprintf("event-api %s (commit: %s, built: %s, %s)", vi.Version, vi.Commit, vi.BuildTime, vi.GoVersion)
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
