// Package homebrew provides Homebrew formula generation and validation for
// prompt-pulse. It produces valid Ruby formula files, bottle configuration,
// tap repository helpers, and macOS launchd service integration.
package homebrew

import "fmt"

// FormulaConfig holds the metadata needed to generate a Homebrew formula.
type FormulaConfig struct {
	// Name is the formula name (e.g. "prompt-pulse").
	Name string

	// Version is the package version string (e.g. "2.0.0").
	Version string

	// Description is a short human-readable summary for the formula desc field.
	Description string

	// Homepage is the project URL.
	Homepage string

	// License is the SPDX license identifier (e.g. "MIT").
	License string

	// GitLabProject is the GitLab project path (e.g. "tinyland/lab/prompt-pulse").
	GitLabProject string

	// HeadBranch is the Git branch used for the head formula (default "main").
	HeadBranch string

	// GoVersion is the minimum Go version required to build (e.g. "1.23").
	GoVersion string

	// LdFlags is a list of linker flags passed to go build.
	LdFlags []string

	// Dependencies lists formula dependencies.
	Dependencies []FormulaDep

	// ShellCompletions enables generation of bash/zsh/fish completions.
	ShellCompletions bool

	// DaemonService enables the Homebrew service block for background operation.
	DaemonService bool
}

// FormulaDep describes a single Homebrew dependency.
type FormulaDep struct {
	// Name is the dependency formula or cask name.
	Name string

	// Type is the dependency category: "build", "runtime", "test", or "optional".
	Type string
}

// DefaultConfig returns a FormulaConfig populated with prompt-pulse v2 defaults.
func DefaultConfig() *FormulaConfig {
	return &FormulaConfig{
		Name:          "prompt-pulse",
		Version:       "2.0.0",
		Description:   "Terminal dashboard with waifu rendering, live data, and TUI mode",
		Homepage:      "https://gitlab.com/tinyland/lab/prompt-pulse",
		License:       "MIT",
		GitLabProject: "tinyland/lab/prompt-pulse",
		HeadBranch:    "main",
		GoVersion:     "1.23",
		LdFlags: []string{
			"-s", "-w",
			"-X main.version=#{version}",
		},
		Dependencies: []FormulaDep{
			{Name: "go", Type: "build"},
		},
		ShellCompletions: true,
		DaemonService:    true,
	}
}

// ValidateConfig checks a FormulaConfig for completeness and returns a list of
// human-readable validation errors. An empty slice means the config is valid.
func ValidateConfig(c *FormulaConfig) []string {
	var errs []string

	if c.Name == "" {
		errs = append(errs, "name is required")
	}
	if c.Version == "" {
		errs = append(errs, "version is required")
	}
	if c.Description == "" {
		errs = append(errs, "description is required")
	}
	if c.Homepage == "" {
		errs = append(errs, "homepage is required")
	}
	if c.License == "" {
		errs = append(errs, "license is required")
	}
	if c.GitLabProject == "" {
		errs = append(errs, "gitlab_project is required")
	}
	if c.HeadBranch == "" {
		errs = append(errs, "head_branch is required")
	}
	if c.GoVersion == "" {
		errs = append(errs, "go_version is required")
	}

	validDepTypes := map[string]bool{
		"build":    true,
		"runtime":  true,
		"test":     true,
		"optional": true,
	}
	for i, dep := range c.Dependencies {
		if dep.Name == "" {
			errs = append(errs, fmt.Sprintf("dependency[%d]: name is required", i))
		}
		if dep.Type != "" && !validDepTypes[dep.Type] {
			errs = append(errs, fmt.Sprintf("dependency[%d]: invalid type %q", i, dep.Type))
		}
	}

	return errs
}
