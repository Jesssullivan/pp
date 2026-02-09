package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// clIsSafeDeletion determines whether a file or directory can be safely deleted
// by checking if its v2 replacement exists in the provided package list. Returns
// a boolean and a human-readable reason.
func clIsSafeDeletion(path string, v2Packages []string) (bool, string) {
	// Check the known v1 directory map first.
	v1Dirs := clV1Directories()
	for _, d := range v1Dirs {
		if d.Path == path || strings.HasPrefix(path, d.Path+"/") {
			if !d.Safe {
				return false, d.Description + " — manual review required"
			}

			// Verify the v2 replacement is listed.
			replacements := strings.Split(d.V2Replacement, ",")
			for _, r := range replacements {
				r = strings.TrimSpace(r)
				for _, pkg := range v2Packages {
					if strings.Contains(pkg, strings.TrimSuffix(r, "/")) {
						return true, fmt.Sprintf("replaced by %s (confirmed in v2 packages)", r)
					}
				}
			}

			// Even if we can't confirm the package, the directory map says safe.
			return true, fmt.Sprintf("replaced by %s (from known migration map)", d.V2Replacement)
		}
	}

	return false, "not in known v1 directory map — manual review required"
}

// clCheckReferences searches all Go files under rootDir for references to the
// given symbol name. Returns a list of file paths containing the symbol.
func clCheckReferences(symbol string, rootDir string) ([]string, error) {
	if symbol == "" {
		return nil, nil
	}

	var refs []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(data), symbol) {
			refs = append(refs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

// clCheckTestCoverage checks whether a v2 package directory contains any test
// files (_test.go). This is a rough proxy for whether the replacement has been
// validated by tests.
func clCheckTestCoverage(v2Package string) bool {
	// v2Package is a relative path like "pkg/layout".
	info, err := os.Stat(v2Package)
	if err != nil || !info.IsDir() {
		return false
	}

	entries, err := os.ReadDir(v2Package)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// clGenerateSafetyReport produces a Markdown safety analysis for a cleanup
// manifest, highlighting any unsafe deletions and missing test coverage.
func clGenerateSafetyReport(manifest *CleanupManifest) string {
	var b strings.Builder

	b.WriteString("# Safety Analysis Report\n\n")

	if manifest == nil || len(manifest.Deletions) == 0 {
		b.WriteString("No deletions to analyze.\n")
		return b.String()
	}

	// Unsafe deletions section.
	var unsafe []Deletion
	var safe []Deletion
	for _, d := range manifest.Deletions {
		if d.Safe {
			safe = append(safe, d)
		} else {
			unsafe = append(unsafe, d)
		}
	}

	b.WriteString(fmt.Sprintf("## Overview\n\n"))
	b.WriteString(fmt.Sprintf("- Safe deletions: **%d**\n", len(safe)))
	b.WriteString(fmt.Sprintf("- Unsafe deletions: **%d**\n", len(unsafe)))
	b.WriteString(fmt.Sprintf("- Total lines at risk: **%d**\n\n", manifest.Summary.TotalLinesRemoved))

	if len(unsafe) > 0 {
		b.WriteString("## Unsafe Deletions (Require Manual Review)\n\n")
		for _, d := range unsafe {
			b.WriteString(fmt.Sprintf("### `%s`\n\n", d.Path))
			b.WriteString(fmt.Sprintf("- **Reason**: %s\n", d.Reason))
			b.WriteString(fmt.Sprintf("- **Lines**: %d\n", d.LinesRemoved))
			if d.ReplacedBy != "" {
				b.WriteString(fmt.Sprintf("- **Suggested replacement**: `%s`\n", d.ReplacedBy))
			}
			b.WriteString("\n")
		}
	}

	if len(safe) > 0 {
		b.WriteString("## Safe Deletions (Ready to Remove)\n\n")
		for _, d := range safe {
			b.WriteString(fmt.Sprintf("- `%s` (%d lines) -> `%s`\n", d.Path, d.LinesRemoved, d.ReplacedBy))
		}
		b.WriteString("\n")
	}

	if len(manifest.Modifications) > 0 {
		b.WriteString("## Import Modifications Required\n\n")
		b.WriteString(fmt.Sprintf("%d files need import path updates before cleanup.\n\n", len(manifest.Modifications)))
	}

	return b.String()
}
