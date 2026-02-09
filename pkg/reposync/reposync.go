// Package reposync provides external repository sync verification and CI
// validation utilities. It manages synchronization of prompt-pulse from its
// monorepo home (crush-dots) to a standalone external repository.
package reposync

import (
	"fmt"
	"strings"
	"time"
)

// SyncConfig describes the mapping between a source monorepo path and a
// target standalone repository.
type SyncConfig struct {
	// SourceRepo is the full repository path (e.g., "gitlab.com/tinyland/lab/crush-dots").
	SourceRepo string

	// SourcePath is the subdirectory within SourceRepo containing the project.
	SourcePath string

	// TargetRepo is the standalone repository that receives synced files.
	TargetRepo string

	// TargetBranch is the branch in TargetRepo that receives synced commits.
	TargetBranch string

	// SyncPaths lists paths (relative to SourcePath) to include in sync.
	SyncPaths []string

	// ExcludePaths lists paths (relative to SourcePath) to exclude from sync.
	ExcludePaths []string

	// CITemplate is the path to the CI template that drives synchronization.
	CITemplate string
}

// SyncStatus captures the current state of synchronization between source
// and target repositories.
type SyncStatus struct {
	// InSync is true when source and target are aligned.
	InSync bool

	// SourceCommit is the latest commit hash on the source side.
	SourceCommit string

	// TargetCommit is the latest commit hash on the target side.
	TargetCommit string

	// DriftFiles lists paths that differ between source and target.
	DriftFiles []string

	// LastSync records when the last successful sync occurred.
	LastSync time.Time
}

// DefaultConfig returns the standard sync configuration for prompt-pulse.
func DefaultConfig() *SyncConfig {
	return &SyncConfig{
		SourceRepo:   "gitlab.com/tinyland/lab/crush-dots",
		SourcePath:   "cmd/prompt-pulse/",
		TargetRepo:   "gitlab.com/tinyland/projects/prompt-pulse",
		TargetBranch: "main",
		SyncPaths: []string{
			"pkg/",
			"go.mod",
			"go.sum",
			"main.go",
			"*.go",
			"docs/",
			"vendor/",
		},
		ExcludePaths: []string{
			"display/",
			"waifu/",
			"collectors/",
			"shell/",
			"config/",
			"cache/",
			"status/",
			"internal/",
			"cmd/",
			"scripts/",
			"tests/",
		},
		CITemplate: "ci/templates/sync-external.yml",
	}
}

// ValidateConfig checks a SyncConfig for common problems and returns a slice
// of human-readable error strings. An empty return means the config is valid.
func ValidateConfig(c *SyncConfig) []string {
	var errs []string

	if c == nil {
		return []string{"config is nil"}
	}

	if strings.TrimSpace(c.SourceRepo) == "" {
		errs = append(errs, "source_repo is required")
	}
	if strings.TrimSpace(c.SourcePath) == "" {
		errs = append(errs, "source_path is required")
	}
	if strings.TrimSpace(c.TargetRepo) == "" {
		errs = append(errs, "target_repo is required")
	}
	if strings.TrimSpace(c.TargetBranch) == "" {
		errs = append(errs, "target_branch is required")
	}
	if len(c.SyncPaths) == 0 {
		errs = append(errs, "sync_paths must not be empty")
	}

	// Warn if source and target are the same repository.
	if c.SourceRepo != "" && c.SourceRepo == c.TargetRepo {
		errs = append(errs, "source_repo and target_repo must differ")
	}

	// Validate that exclude paths don't overlap with explicit sync paths.
	for _, ep := range c.ExcludePaths {
		for _, sp := range c.SyncPaths {
			if ep == sp {
				errs = append(errs, fmt.Sprintf("path %q appears in both sync_paths and exclude_paths", ep))
			}
		}
	}

	return errs
}
