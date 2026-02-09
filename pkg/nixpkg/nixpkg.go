// Package nixpkg provides Nix packaging helpers and validation for prompt-pulse.
// It generates buildGoModule derivations, overlays, and dev shells, and validates
// binary artifacts produced by Nix builds.
package nixpkg

// PackageMeta holds Nix package metadata for the prompt-pulse derivation.
type PackageMeta struct {
	// Name is the Nix package name (pname).
	Name string

	// Version is the package version string.
	Version string

	// Description is a short package description for Nix meta.
	Description string

	// License is the SPDX license identifier (e.g. "MIT").
	License string

	// Homepage is the upstream project URL.
	Homepage string

	// MainPackage is the Go import path relative to the module root.
	MainPackage string

	// Architectures lists Nix platform identifiers.
	Architectures []string

	// Dependencies lists Nix package names required at build time.
	Dependencies []string
}

// DefaultMeta returns PackageMeta populated with prompt-pulse v2 defaults.
func DefaultMeta() *PackageMeta {
	return &PackageMeta{
		Name:        "prompt-pulse",
		Version:     "2.0.0",
		Description: "Terminal dashboard with waifu rendering, live data, and TUI mode",
		License:     "MIT",
		Homepage:    "https://gitlab.com/tinyland/lab/prompt-pulse",
		MainPackage: "./cmd/prompt-pulse",
		Architectures: []string{
			"aarch64-darwin",
			"x86_64-linux",
			"aarch64-linux",
		},
		Dependencies: []string{"go_1_23"},
	}
}

// ValidateMeta checks the given PackageMeta for completeness and returns
// a list of human-readable validation errors. An empty slice means valid.
func ValidateMeta(m *PackageMeta) []string {
	var errs []string

	if m.Name == "" {
		errs = append(errs, "name is required")
	}
	if m.Version == "" {
		errs = append(errs, "version is required")
	}
	if m.Description == "" {
		errs = append(errs, "description is required")
	}
	if m.License == "" {
		errs = append(errs, "license is required")
	}
	if m.Homepage == "" {
		errs = append(errs, "homepage is required")
	}
	if m.MainPackage == "" {
		errs = append(errs, "main package path is required")
	}
	if len(m.Architectures) == 0 {
		errs = append(errs, "at least one architecture is required")
	}
	if len(m.Dependencies) == 0 {
		errs = append(errs, "at least one dependency is required")
	}

	return errs
}
