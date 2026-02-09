package reposync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncManifest records the complete set of files involved in a sync operation.
type SyncManifest struct {
	// Version is the manifest format version.
	Version string `json:"version"`

	// SourceCommit identifies the source revision.
	SourceCommit string `json:"source_commit"`

	// Files lists every file considered during sync.
	Files []SyncFile `json:"files"`

	// GeneratedAt records when the manifest was created.
	GeneratedAt time.Time `json:"generated_at"`
}

// SyncFile describes a single file's role in the sync.
type SyncFile struct {
	// SourcePath is the path relative to the source root.
	SourcePath string `json:"source_path"`

	// TargetPath is the destination path in the target repo.
	TargetPath string `json:"target_path"`

	// Hash is the SHA-256 hex digest of the file content.
	Hash string `json:"hash"`

	// Action describes what happens to this file: "copy", "rewrite", or "exclude".
	Action string `json:"action"`
}

// rsGenerateManifest walks rootDir and builds a SyncManifest by applying the
// include/exclude rules from config.
func rsGenerateManifest(config *SyncConfig, rootDir string) (*SyncManifest, error) {
	if config == nil {
		return nil, fmt.Errorf("config must not be nil")
	}

	manifest := &SyncManifest{
		Version:     "1",
		GeneratedAt: time.Now().UTC(),
	}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		// Normalise separators.
		relPath = filepath.ToSlash(relPath)

		action := rsClassifyFile(relPath, config.SyncPaths, config.ExcludePaths)
		if action == "exclude" {
			manifest.Files = append(manifest.Files, SyncFile{
				SourcePath: relPath,
				TargetPath: relPath,
				Action:     "exclude",
			})
			return nil
		}

		// Compute hash.
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))

		// Determine if rewrite is needed.
		if action == "copy" && (relPath == "go.mod" || strings.HasSuffix(relPath, ".go")) {
			action = "rewrite"
		}

		manifest.Files = append(manifest.Files, SyncFile{
			SourcePath: relPath,
			TargetPath: relPath,
			Hash:       hash,
			Action:     action,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return manifest, nil
}

// rsClassifyFile decides whether a file should be copied, rewritten, or excluded.
func rsClassifyFile(relPath string, include, exclude []string) string {
	// Check exclusion first.
	for _, ex := range exclude {
		if rsPathMatch(relPath, ex) {
			return "exclude"
		}
	}

	// Check inclusion.
	for _, inc := range include {
		if rsPathMatch(relPath, inc) {
			return "copy"
		}
	}

	// Files not matching any include pattern are excluded by default.
	return "exclude"
}

// rsPathMatch checks if a relative path matches a pattern. Patterns ending
// in "/" match directory prefixes. Patterns starting with "*." match by
// file extension. Otherwise, exact match is required.
func rsPathMatch(path, pattern string) bool {
	// Directory prefix match.
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern) || path+"/" == pattern
	}

	// Glob-style extension match (e.g., "*.go").
	if strings.HasPrefix(pattern, "*.") {
		ext := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(path, ext)
	}

	// Exact match.
	return path == pattern
}

// rsRenderManifestJSON serializes a SyncManifest to indented JSON.
func rsRenderManifestJSON(manifest *SyncManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest must not be nil")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling manifest: %w", err)
	}
	return string(data), nil
}

// rsRenderManifestMarkdown generates a human-readable markdown summary.
func rsRenderManifestMarkdown(manifest *SyncManifest) string {
	if manifest == nil {
		return "No manifest available."
	}

	var b strings.Builder
	b.WriteString("# Sync Manifest\n\n")
	b.WriteString(fmt.Sprintf("- **Version**: %s\n", manifest.Version))
	b.WriteString(fmt.Sprintf("- **Source Commit**: %s\n", manifest.SourceCommit))
	b.WriteString(fmt.Sprintf("- **Generated**: %s\n", manifest.GeneratedAt.Format(time.RFC3339)))

	// Count actions.
	counts := map[string]int{}
	for _, f := range manifest.Files {
		counts[f.Action]++
	}
	b.WriteString(fmt.Sprintf("- **Total Files**: %d\n", len(manifest.Files)))
	for _, action := range []string{"copy", "rewrite", "exclude"} {
		if c, ok := counts[action]; ok {
			b.WriteString(fmt.Sprintf("  - %s: %d\n", action, c))
		}
	}

	b.WriteString("\n## Files\n\n")
	b.WriteString("| Source | Target | Action | Hash |\n")
	b.WriteString("|--------|--------|--------|------|\n")
	for _, f := range manifest.Files {
		hash := f.Hash
		if len(hash) > 12 {
			hash = hash[:12] + "..."
		}
		if hash == "" {
			hash = "-"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.SourcePath, f.TargetPath, f.Action, hash))
	}

	return b.String()
}

// rsFilterPaths applies include and exclude rules to a list of paths and
// returns only the paths that match at least one include pattern and do not
// match any exclude pattern.
func rsFilterPaths(allPaths []string, include, exclude []string) []string {
	var result []string
	for _, p := range allPaths {
		if rsClassifyFile(p, include, exclude) != "exclude" {
			result = append(result, p)
		}
	}
	return result
}
