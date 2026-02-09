// Package cleanup provides dead code analysis and cleanup validation for the
// v1-to-v2 migration. It scans the codebase, identifies v1 code that has been
// replaced by v2 packages, and generates a cleanup manifest describing what
// can safely be removed. This package analyzes only — it never deletes files.
package cleanup

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// CleanupManifest describes the full set of deletions and modifications needed
// to remove v1 dead code from the repository.
type CleanupManifest struct {
	Deletions     []Deletion
	Modifications []Modification
	Summary       ManifestSummary
}

// Deletion represents a single file or directory that can be removed.
type Deletion struct {
	Path         string
	Reason       string
	Safe         bool
	LinesRemoved int
	ReplacedBy   string
}

// Modification represents a file that needs import path updates rather than
// full deletion.
type Modification struct {
	Path        string
	Description string
	OldImport   string
	NewImport   string
}

// ManifestSummary aggregates high-level statistics about the cleanup.
type ManifestSummary struct {
	TotalDeletions     int
	TotalModifications int
	TotalLinesRemoved  int
	SafeDeletions      int
	UnsafeDeletions    int
}

// Analyze scans the codebase rooted at rootDir and generates a CleanupManifest
// describing v1 code that can be removed or updated.
func Analyze(rootDir string) (*CleanupManifest, error) {
	manifest := &CleanupManifest{}

	// Scan for v1 directories that exist on disk.
	v1Dirs := clV1Directories()
	for _, d := range v1Dirs {
		dirPath := filepath.Join(rootDir, d.Path)
		files, err := clScanDirectory(dirPath)
		if err != nil {
			// Directory doesn't exist — skip.
			continue
		}

		totalLines := 0
		for _, f := range files {
			totalLines += f.Lines
		}

		manifest.Deletions = append(manifest.Deletions, Deletion{
			Path:         d.Path,
			Reason:       d.Description,
			Safe:         d.Safe,
			LinesRemoved: totalLines,
			ReplacedBy:   d.V2Replacement,
		})
	}

	// Scan for import modifications needed.
	importRefs, err := clFindV1Imports(rootDir)
	if err == nil {
		for _, ref := range importRefs {
			if ref.V2Replacement != "" {
				manifest.Modifications = append(manifest.Modifications, Modification{
					Path:        ref.FilePath,
					Description: fmt.Sprintf("Update import from v1 to v2 package"),
					OldImport:   ref.ImportPath,
					NewImport:   ref.V2Replacement,
				})
			}
		}
	}

	// Compute summary.
	manifest.Summary = clComputeSummary(manifest)

	return manifest, nil
}

// clComputeSummary derives aggregate statistics from the manifest contents.
func clComputeSummary(m *CleanupManifest) ManifestSummary {
	s := ManifestSummary{
		TotalDeletions:     len(m.Deletions),
		TotalModifications: len(m.Modifications),
	}
	for _, d := range m.Deletions {
		s.TotalLinesRemoved += d.LinesRemoved
		if d.Safe {
			s.SafeDeletions++
		} else {
			s.UnsafeDeletions++
		}
	}
	return s
}

// RenderMarkdown renders the manifest as a Markdown report suitable for review.
func (m *CleanupManifest) RenderMarkdown() string {
	var b strings.Builder

	b.WriteString("# Cleanup Manifest\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- **Total Deletions**: %d (%d safe, %d unsafe)\n",
		m.Summary.TotalDeletions, m.Summary.SafeDeletions, m.Summary.UnsafeDeletions))
	b.WriteString(fmt.Sprintf("- **Total Modifications**: %d\n", m.Summary.TotalModifications))
	b.WriteString(fmt.Sprintf("- **Total Lines Removed**: %d\n", m.Summary.TotalLinesRemoved))

	if len(m.Deletions) > 0 {
		b.WriteString("\n## Deletions\n\n")
		// Sort: safe first, then by path.
		sorted := make([]Deletion, len(m.Deletions))
		copy(sorted, m.Deletions)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].Safe != sorted[j].Safe {
				return sorted[i].Safe
			}
			return sorted[i].Path < sorted[j].Path
		})

		for _, d := range sorted {
			safeStr := "SAFE"
			if !d.Safe {
				safeStr = "UNSAFE"
			}
			b.WriteString(fmt.Sprintf("### `%s` [%s]\n\n", d.Path, safeStr))
			b.WriteString(fmt.Sprintf("- **Reason**: %s\n", d.Reason))
			b.WriteString(fmt.Sprintf("- **Lines**: %d\n", d.LinesRemoved))
			if d.ReplacedBy != "" {
				b.WriteString(fmt.Sprintf("- **Replaced by**: `%s`\n", d.ReplacedBy))
			}
			b.WriteString("\n")
		}
	}

	if len(m.Modifications) > 0 {
		b.WriteString("## Modifications\n\n")
		b.WriteString("| File | Old Import | New Import |\n")
		b.WriteString("|------|-----------|------------|\n")
		for _, mod := range m.Modifications {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` |\n", mod.Path, mod.OldImport, mod.NewImport))
		}
	}

	return b.String()
}

// RenderScript generates a shell script with rm -rf commands for the deletions
// marked as safe. The script is meant for human review, NOT automatic execution.
func (m *CleanupManifest) RenderScript() string {
	var b strings.Builder

	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("# Generated cleanup script — REVIEW BEFORE RUNNING\n")
	b.WriteString("# This script removes v1 code that has been replaced by v2 packages.\n")
	b.WriteString("#\n")
	b.WriteString(fmt.Sprintf("# Safe deletions:   %d\n", m.Summary.SafeDeletions))
	b.WriteString(fmt.Sprintf("# Unsafe deletions: %d (commented out)\n", m.Summary.UnsafeDeletions))
	b.WriteString(fmt.Sprintf("# Lines removed:    %d\n", m.Summary.TotalLinesRemoved))
	b.WriteString("\nset -euo pipefail\n\n")

	// Sort: safe first, then by path for stable output.
	sorted := make([]Deletion, len(m.Deletions))
	copy(sorted, m.Deletions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Safe != sorted[j].Safe {
			return sorted[i].Safe
		}
		return sorted[i].Path < sorted[j].Path
	})

	for _, d := range sorted {
		if d.ReplacedBy != "" {
			b.WriteString(fmt.Sprintf("# %s -> %s (%d lines)\n", d.Path, d.ReplacedBy, d.LinesRemoved))
		} else {
			b.WriteString(fmt.Sprintf("# %s (%d lines)\n", d.Path, d.LinesRemoved))
		}
		if d.Safe {
			b.WriteString(fmt.Sprintf("rm -rf %q\n\n", d.Path))
		} else {
			b.WriteString(fmt.Sprintf("# rm -rf %q  # UNSAFE: %s\n\n", d.Path, d.Reason))
		}
	}

	return b.String()
}
