package cleanup

import (
	"os"
	"path/filepath"
	"strings"
)

// ImportRef describes a single import statement in a file that references a v1 package.
type ImportRef struct {
	FilePath      string
	ImportPath    string
	V2Replacement string
	Line          int
}

// clV1ImportMap returns the mapping from v1 import prefixes to their v2 replacements.
func clV1ImportMap() map[string]string {
	const base = "gitlab.com/tinyland/lab/prompt-pulse"
	return map[string]string{
		base + "/display/banner":  base + "/pkg/layout",
		base + "/display/color":   base + "/pkg/theme",
		base + "/display/layout":  base + "/pkg/layout",
		base + "/display/render":  base + "/pkg/image",
		base + "/display/starship": base + "/pkg/starship",
		base + "/display/tui":     base + "/pkg/tui",
		base + "/display/widgets": base + "/pkg/widgets",
		base + "/waifu":           base + "/pkg/waifu",
		base + "/collectors":      base + "/pkg/collectors",
		base + "/shell":           base + "/pkg/shell",
		base + "/config":          base + "/pkg/config",
		base + "/cache":           base + "/pkg/cache",
		base + "/status":          base + "/pkg/widgets",
		base + "/internal/format": base + "/pkg/data",
	}
}

// clFindV1Imports scans all Go files under rootDir and finds import statements
// that reference v1 packages. For each match, it records the file path, import
// path, suggested v2 replacement, and line number.
func clFindV1Imports(rootDir string) ([]ImportRef, error) {
	v1Map := clV1ImportMap()
	var refs []ImportRef

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			// Skip vendor and .git directories.
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fileRefs, scanErr := clScanFileImports(path, v1Map)
		if scanErr != nil {
			return nil // Skip files that can't be read.
		}
		refs = append(refs, fileRefs...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

// clScanFileImports reads a single file and checks each import against the v1 map.
func clScanFileImports(path string, v1Map map[string]string) ([]ImportRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var refs []ImportRef

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for import lines containing a quoted path.
		if !strings.Contains(trimmed, `"`) {
			continue
		}

		// Extract the import path from between quotes.
		importPath := clExtractQuotedImport(trimmed)
		if importPath == "" {
			continue
		}

		// Check against every v1 prefix.
		for v1Prefix, v2Replacement := range v1Map {
			if importPath == v1Prefix || strings.HasPrefix(importPath, v1Prefix+"/") {
				refs = append(refs, ImportRef{
					FilePath:      path,
					ImportPath:    importPath,
					V2Replacement: v2Replacement,
					Line:          i + 1,
				})
				break
			}
		}
	}
	return refs, nil
}

// clExtractQuotedImport extracts the first double-quoted string from a line.
func clExtractQuotedImport(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// clSuggestImportUpdate returns a human-readable suggestion for updating a v1
// import to its v2 replacement.
func clSuggestImportUpdate(ref ImportRef) string {
	if ref.V2Replacement == "" {
		return "no v2 replacement available"
	}
	return strings.Replace(
		ref.ImportPath,
		ref.ImportPath,
		ref.V2Replacement,
		1,
	) + " (was " + ref.ImportPath + ")"
}

// clFindOrphanedImports scans all Go files under rootDir and finds import paths
// that reference packages within the module but have no corresponding directory
// on disk. These are imports pointing to deleted or missing packages.
func clFindOrphanedImports(rootDir string) ([]string, error) {
	const moduleBase = "gitlab.com/tinyland/lab/prompt-pulse"

	// Collect all Go files.
	var allFiles []string
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
		if strings.HasSuffix(path, ".go") {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Collect all unique internal imports and check if their directory exists.
	seen := make(map[string]bool)
	var orphans []string

	for _, fpath := range allFiles {
		parsed, parseErr := clParseGoFile(fpath)
		if parseErr != nil {
			continue
		}
		for _, imp := range parsed.Imports {
			if !strings.HasPrefix(imp, moduleBase) {
				continue
			}
			if seen[imp] {
				continue
			}
			seen[imp] = true

			// Convert import path to directory path.
			relPath := strings.TrimPrefix(imp, moduleBase+"/")
			dirPath := filepath.Join(rootDir, relPath)
			if _, statErr := os.Stat(dirPath); os.IsNotExist(statErr) {
				orphans = append(orphans, imp)
			}
		}
	}

	return orphans, nil
}
