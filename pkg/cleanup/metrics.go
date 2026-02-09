package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CodeMetrics holds aggregate code statistics for v1 and v2 codebases.
type CodeMetrics struct {
	V1Lines          int
	V2Lines          int
	V1Files          int
	V2Files          int
	V1Packages       int
	V2Packages       int
	DuplicationRatio float64
	V2TestCount      int
}

// clComputeMetrics scans rootDir and computes line counts, file counts, and
// package counts for both the v1 top-level directories and the v2 pkg/ tree.
func clComputeMetrics(rootDir string) (*CodeMetrics, error) {
	m := &CodeMetrics{}

	// Scan v1 directories.
	v1Pkgs := make(map[string]bool)
	v1Dirs := clV1Directories()
	for _, d := range v1Dirs {
		dirPath := filepath.Join(rootDir, d.Path)
		files, err := clScanDirectory(dirPath)
		if err != nil {
			continue
		}
		for _, f := range files {
			m.V1Files++
			m.V1Lines += f.Lines
			v1Pkgs[f.Package] = true
		}
	}
	m.V1Packages = len(v1Pkgs)

	// Scan v2 pkg/ directory.
	pkgDir := filepath.Join(rootDir, "pkg")
	v2Pkgs := make(map[string]bool)
	v2Files, err := clScanDirectory(pkgDir)
	if err == nil {
		for _, f := range v2Files {
			m.V2Files++
			m.V2Lines += f.Lines
			v2Pkgs[f.Package] = true
			if f.IsTest {
				m.V2TestCount++
			}
		}
	}
	m.V2Packages = len(v2Pkgs)

	// Compute duplication ratio.
	v1NonTest := clFilterNonTest(clCollectV1Files(rootDir))
	v2NonTest := clFilterNonTest(v2Files)
	m.DuplicationRatio = clComputeDuplicationRatio(v1NonTest, v2NonTest)

	return m, nil
}

// clCollectV1Files gathers all v1 FileInfo entries across known v1 directories.
func clCollectV1Files(rootDir string) []FileInfo {
	var all []FileInfo
	for _, d := range clV1Directories() {
		dirPath := filepath.Join(rootDir, d.Path)
		files, err := clScanDirectory(dirPath)
		if err != nil {
			continue
		}
		all = append(all, files...)
	}
	return all
}

// clFilterNonTest returns only non-test files from the input.
func clFilterNonTest(files []FileInfo) []FileInfo {
	var out []FileInfo
	for _, f := range files {
		if !f.IsTest {
			out = append(out, f)
		}
	}
	return out
}

// clEstimateDebt returns a human-readable summary of the technical debt
// implied by the metrics. This helps justify the cleanup to stakeholders.
func clEstimateDebt(metrics *CodeMetrics) string {
	if metrics == nil {
		return "no metrics available"
	}

	var parts []string

	if metrics.V1Lines > 0 {
		parts = append(parts, fmt.Sprintf("%d lines of v1 code can be removed", metrics.V1Lines))
	}
	if metrics.V1Files > 0 {
		parts = append(parts, fmt.Sprintf("%d v1 files across %d packages", metrics.V1Files, metrics.V1Packages))
	}
	if metrics.DuplicationRatio > 0 {
		parts = append(parts, fmt.Sprintf("%.0f%% estimated duplication between v1 and v2", metrics.DuplicationRatio*100))
	}
	if metrics.V2TestCount > 0 {
		parts = append(parts, fmt.Sprintf("%d v2 test files validate replacements", metrics.V2TestCount))
	}

	if len(parts) == 0 {
		return "codebase is clean — no v1 debt detected"
	}

	return "Tech debt summary: " + strings.Join(parts, "; ") + "."
}

// clComputeDuplicationRatio estimates how much of the v1 code is duplicated by
// v2 code. Uses package-name overlap and filename overlap as heuristics.
// Returns a value between 0.0 (no overlap) and 1.0 (complete overlap).
func clComputeDuplicationRatio(v1Files, v2Files []FileInfo) float64 {
	if len(v1Files) == 0 {
		return 0.0
	}

	// Build sets of v2 package names and base filenames.
	v2Pkgs := make(map[string]bool)
	v2Bases := make(map[string]bool)
	for _, f := range v2Files {
		v2Pkgs[f.Package] = true
		v2Bases[filepath.Base(f.Path)] = true
	}

	// Count how many v1 files have a matching v2 counterpart.
	matches := 0
	for _, f := range v1Files {
		base := filepath.Base(f.Path)
		if v2Bases[base] || v2Pkgs[f.Package] {
			matches++
		}
	}

	return float64(matches) / float64(len(v1Files))
}

// clCountDirectoryLines counts total lines across all .go files in a directory.
func clCountDirectoryLines(dir string) (int, error) {
	total := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		lines, lineErr := clCountLines(path)
		if lineErr != nil {
			return nil
		}
		total += lines
		return nil
	})
	return total, err
}
