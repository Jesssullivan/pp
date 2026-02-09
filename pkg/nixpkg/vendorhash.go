package nixpkg

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VendorInfo holds metadata about the Go module's vendored dependencies,
// used when generating the vendorHash field in a Nix derivation.
type VendorInfo struct {
	// Hash is the vendor hash in Nix SRI format (e.g. "sha256-XXXX...").
	Hash string

	// GoVersion is the Go version declared in go.mod.
	GoVersion string

	// NumDeps is the number of dependency entries found in go.sum.
	NumDeps int

	// Timestamp records when this info was computed.
	Timestamp time.Time
}

// npComputeVendorInfo reads go.mod and go.sum from modDir to produce VendorInfo.
// It extracts the Go version from go.mod and counts dependency lines in go.sum.
// The Hash field is left empty; it must be filled by a Nix build or prefetch step.
func npComputeVendorInfo(modDir string) (*VendorInfo, error) {
	goVersion, err := npParseGoVersion(filepath.Join(modDir, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	numDeps, err := npCountDeps(filepath.Join(modDir, "go.sum"))
	if err != nil {
		return nil, fmt.Errorf("count deps in go.sum: %w", err)
	}

	return &VendorInfo{
		GoVersion: goVersion,
		NumDeps:   numDeps,
		Timestamp: time.Now(),
	}, nil
}

// npFormatNixHash formats a raw SHA-256 hash string into the Nix SRI format.
// If the hash already has the "sha256-" prefix it is returned unchanged.
func npFormatNixHash(hash string) string {
	if strings.HasPrefix(hash, "sha256-") {
		return hash
	}
	return "sha256-" + hash
}

// npValidateVendorDir checks that the vendor directory at vendorDir is
// structurally valid: it must exist, contain modules.txt, and have no
// empty subdirectories at the top level.
func npValidateVendorDir(vendorDir string) error {
	info, err := os.Stat(vendorDir)
	if err != nil {
		return fmt.Errorf("vendor directory not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("vendor path is not a directory: %s", vendorDir)
	}

	modulesTxt := filepath.Join(vendorDir, "modules.txt")
	if _, err := os.Stat(modulesTxt); err != nil {
		return fmt.Errorf("modules.txt not found in vendor directory: %w", err)
	}

	entries, err := os.ReadDir(vendorDir)
	if err != nil {
		return fmt.Errorf("read vendor directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subPath := filepath.Join(vendorDir, entry.Name())
		subEntries, err := os.ReadDir(subPath)
		if err != nil {
			return fmt.Errorf("read vendor subdirectory %s: %w", entry.Name(), err)
		}
		if len(subEntries) == 0 {
			return fmt.Errorf("empty vendor subdirectory: %s", entry.Name())
		}
	}

	return nil
}

// npParseGoVersion extracts the "go X.Y" version from a go.mod file.
func npParseGoVersion(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "go ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no go directive found in %s", goModPath)
}

// npCountDeps counts non-empty, non-comment lines in go.sum.
func npCountDeps(goSumPath string) (int, error) {
	f, err := os.Open(goSumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "//") {
			count++
		}
	}
	return count, scanner.Err()
}
