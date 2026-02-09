package nixpkg

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildResult holds the outcome of validating a Nix-built binary.
type BuildResult struct {
	// Success indicates whether all validation checks passed.
	Success bool

	// BinaryPath is the absolute path to the validated binary.
	BinaryPath string

	// BinarySize is the file size in bytes.
	BinarySize int64

	// Architectures lists detected architectures from the binary.
	Architectures []string

	// Errors collects validation failure messages.
	Errors []string
}

// npValidateBinary checks that the binary at binaryPath exists, is
// executable, and reports its size and detected architecture.
func npValidateBinary(binaryPath string) (*BuildResult, error) {
	result := &BuildResult{
		BinaryPath: binaryPath,
	}

	info, err := os.Stat(binaryPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("binary not found: %v", err))
		return result, nil
	}

	if info.IsDir() {
		result.Errors = append(result.Errors, "path is a directory, not a binary")
		return result, nil
	}

	result.BinarySize = info.Size()

	if info.Size() == 0 {
		result.Errors = append(result.Errors, "binary is empty (0 bytes)")
		return result, nil
	}

	// Check executable bit.
	if info.Mode()&0111 == 0 {
		result.Errors = append(result.Errors, "binary is not executable")
	}

	// Detect architecture.
	arch, err := npDetectArch(binaryPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("arch detection failed: %v", err))
	} else {
		result.Architectures = []string{arch}
	}

	result.Success = len(result.Errors) == 0
	return result, nil
}

// npValidateCompletions checks that shell completion scripts exist in
// the given directory for bash, zsh, and fish.
func npValidateCompletions(binaryDir string) error {
	completions := map[string]string{
		"bash": "prompt-pulse.bash",
		"zsh":  "_prompt-pulse",
		"fish": "prompt-pulse.fish",
	}

	var missing []string
	for shell, filename := range completions {
		path := filepath.Join(binaryDir, filename)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, shell)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing completions for: %s", strings.Join(missing, ", "))
	}
	return nil
}

// npCheckDeps reads go.mod at goModPath and returns the list of direct
// (non-indirect) module dependencies.
func npCheckDeps(goModPath string) ([]string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("open go.mod: %w", err)
	}
	defer f.Close()

	var deps []string
	inRequire := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "require (") || line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			// Skip indirect dependencies.
			if strings.Contains(line, "// indirect") {
				continue
			}
			// Skip empty lines and comments.
			if line == "" || strings.HasPrefix(line, "//") {
				continue
			}
			// Extract module path (first field).
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				deps = append(deps, parts[0])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan go.mod: %w", err)
	}

	return deps, nil
}

// npDetectArch uses the `file` command to detect the architecture of a
// compiled binary, returning a Nix-style platform string.
func npDetectArch(binaryPath string) (string, error) {
	out, err := exec.Command("file", binaryPath).Output()
	if err != nil {
		return "", fmt.Errorf("run file command: %w", err)
	}

	output := string(out)
	return npParseFileOutput(output)
}

// npParseFileOutput parses the output of the `file` command and returns
// a Nix-style platform string.
func npParseFileOutput(output string) (string, error) {
	lower := strings.ToLower(output)

	switch {
	case strings.Contains(lower, "mach-o") && strings.Contains(lower, "arm64"):
		return "aarch64-darwin", nil
	case strings.Contains(lower, "mach-o") && strings.Contains(lower, "x86_64"):
		return "x86_64-darwin", nil
	case strings.Contains(lower, "elf") && strings.Contains(lower, "aarch64"):
		return "aarch64-linux", nil
	case strings.Contains(lower, "elf") && strings.Contains(lower, "x86-64"):
		return "x86_64-linux", nil
	case strings.Contains(lower, "elf") && strings.Contains(lower, "arm"):
		return "armv7l-linux", nil
	default:
		return "", fmt.Errorf("unknown architecture in file output: %s", strings.TrimSpace(output))
	}
}
