package cleanup

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo holds parsed metadata about a single Go source file.
type FileInfo struct {
	Path            string
	Package         string
	Lines           int
	Imports         []string
	ExportedSymbols []string
	IsTest          bool
}

// DuplicateInfo describes a v1 file that has a likely v2 replacement.
type DuplicateInfo struct {
	V1Path    string
	V2Path    string
	V1Package string
	V2Package string
	// Confidence from 0.0 to 1.0 indicating how likely the v2 file replaces v1.
	Confidence float64
	Reason     string
}

// clScanDirectory recursively scans dir for .go files and returns parsed FileInfo
// for each. Returns an error if dir does not exist or cannot be read.
func clScanDirectory(dir string) ([]FileInfo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "scan", Path: dir, Err: os.ErrInvalid}
	}

	var files []FileInfo
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		parsed, parseErr := clParseGoFile(path)
		if parseErr != nil {
			// Skip unparseable files.
			return nil
		}
		files = append(files, *parsed)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// clParseGoFile parses a single Go source file and extracts its package name,
// import paths, exported symbol names, and line count.
func clParseGoFile(path string) (*FileInfo, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		// Fall back to partial parse for package name.
		f, err = parser.ParseFile(fset, path, src, parser.PackageClauseOnly)
		if err != nil {
			return nil, err
		}
	}

	fi := &FileInfo{
		Path:    path,
		Package: f.Name.Name,
		Lines:   clCountLinesBytes(src),
		IsTest:  strings.HasSuffix(path, "_test.go"),
	}

	// Extract imports.
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		fi.Imports = append(fi.Imports, importPath)
	}

	// For exported symbols we need a fuller parse.
	fullFile, fullErr := parser.ParseFile(token.NewFileSet(), path, src, 0)
	if fullErr == nil {
		fi.ExportedSymbols = clExtractExported(fullFile)
	}

	return fi, nil
}

// clExtractExported walks an AST and collects names of exported declarations.
func clExtractExported(f *ast.File) []string {
	var exported []string
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				exported = append(exported, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						exported = append(exported, s.Name.Name)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							exported = append(exported, name.Name)
						}
					}
				}
			}
		}
	}
	return exported
}

// clCountLines counts the number of newline-delimited lines in a file.
func clCountLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return clCountLinesBytes(data), nil
}

// clCountLinesBytes counts lines in a byte slice. An empty file has 0 lines.
// A file with content but no trailing newline still counts its last line.
func clCountLinesBytes(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	// If the file doesn't end with a newline, count the last partial line.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		count++
	}
	return count
}

// clFindDuplicates compares v1 and v2 file lists and identifies likely replacements
// based on file name similarity and package names.
func clFindDuplicates(v1Files, v2Files []FileInfo) []DuplicateInfo {
	var dupes []DuplicateInfo

	// Build lookup maps for v2 files.
	v2ByBase := make(map[string][]FileInfo)
	v2ByPkg := make(map[string][]FileInfo)
	for _, f := range v2Files {
		base := filepath.Base(f.Path)
		v2ByBase[base] = append(v2ByBase[base], f)
		v2ByPkg[f.Package] = append(v2ByPkg[f.Package], f)
	}

	for _, v1 := range v1Files {
		if v1.IsTest {
			continue
		}

		base := filepath.Base(v1.Path)
		bestConfidence := 0.0
		var bestMatch *FileInfo
		var bestReason string

		// Check for exact filename match.
		if matches, ok := v2ByBase[base]; ok {
			for i := range matches {
				m := &matches[i]
				conf := 0.7
				reason := "same filename"
				if m.Package == v1.Package {
					conf = 0.9
					reason = "same filename and package"
				}
				if conf > bestConfidence {
					bestConfidence = conf
					bestMatch = m
					bestReason = reason
				}
			}
		}

		// Check for same package name (lower confidence).
		if matches, ok := v2ByPkg[v1.Package]; ok && bestConfidence < 0.5 {
			for i := range matches {
				m := &matches[i]
				conf := 0.5
				reason := "same package name"
				if conf > bestConfidence {
					bestConfidence = conf
					bestMatch = m
					bestReason = reason
				}
			}
		}

		if bestMatch != nil && bestConfidence >= 0.5 {
			dupes = append(dupes, DuplicateInfo{
				V1Path:     v1.Path,
				V2Path:     bestMatch.Path,
				V1Package:  v1.Package,
				V2Package:  bestMatch.Package,
				Confidence: bestConfidence,
				Reason:     bestReason,
			})
		}
	}

	return dupes
}
