package docs

import (
	"fmt"
	"strings"
)

// Changelog holds release notes for all versions.
type Changelog struct {
	// Releases lists all releases, newest first.
	Releases []Release
}

// Release documents a single version release.
type Release struct {
	// Version is the semver version string.
	Version string

	// Date is the release date in YYYY-MM-DD format.
	Date string

	// Added lists new features.
	Added []string

	// Changed lists modifications to existing features.
	Changed []string

	// Fixed lists bug fixes.
	Fixed []string

	// Removed lists removed features.
	Removed []string
}

// dcGenerateV2Changelog builds the v2.0.0 changelog from embedded knowledge.
func dcGenerateV2Changelog() *Changelog {
	return &Changelog{
		Releases: []Release{
			{
				Version: "2.0.0",
				Date:    "2026-02-09",
				Added: []string{
					"Bubbletea v2 full-screen TUI mode with interactive widget grid",
					"Cassowary constraint-based layout engine for terminal widget positioning",
					"Six named color themes: default, gruvbox, nord, catppuccin, dracula, tokyo-night",
					"Four layout presets: dashboard, minimal, ops, billing",
					"Ksh shell integration with KEYBD trap and PS1 command substitution",
					"Emacs integration via elisp helpers and Unix socket queries",
					"Kitty Unicode placeholder protocol for crisp image rendering",
					"Performance benchmark framework with render latency and memory tracking",
					"Cross-platform abstractions for macOS and Linux (APFS-aware disk reporting)",
					"Multi-protocol image rendering: Kitty, iTerm2, Sixel, half-blocks with auto-detection",
					"Cloud billing collector for Civo and DigitalOcean spend tracking",
					"Kubernetes multi-context status collector with namespace filtering",
					"Starship prompt segment integration for inline status display",
					"Configuration migration tool from v1 flat format to v2 nested TOML",
					"Nix packaging with buildGoModule derivation and overlay generation",
					"Homebrew formula generation and tap management",
					"Documentation generator for architecture docs, config reference, and man pages",
					"Terminal test harness with virtual terminal emulation",
					"Shell integration test framework with script execution and validation",
					"Integration test utilities with daemon lifecycle management",
					"Daemon Unix socket IPC for background data collection",
					"Waifu image caching with TTL expiry and disk size limits",
					"Reusable Bubbletea components: sparklines, gauges, tables, bordered panels",
					"Vim-style navigation (h/j/k/l) and mouse support in TUI mode",
					"Adaptive banner width modes: compact, standard, wide, ultra-wide",
					"Tab completions for Bash (complete -C), Zsh (compdef), and Fish",
					"Pre-rendered banner cache for sub-millisecond shell startup display",
					"Data retention controls with configurable time-series expiry",
					"Environment variable overrides for sensitive config values",
				},
				Changed: []string{
					"Configuration format migrated from flat key=value to nested TOML tables",
					"Image rendering upgraded from single-protocol to multi-protocol with auto-detection",
					"Shell integration expanded from Bash/Zsh to include Fish and Ksh",
					"Architecture refactored from monolithic single-package to 29 focused packages",
					"Layout engine replaced with Cassowary constraint solver for flexible positioning",
					"Theme system upgraded from hardcoded colors to named palette registry",
					"Collector framework redesigned with configurable intervals and graceful degradation",
					"Cache system rewritten with disk persistence and session isolation",
				},
				Fixed: []string{
					"APFS disk usage reporting now uses actual block allocation instead of apparent size",
					"Terminal capability detection works correctly inside tmux and screen sessions",
					"Image rendering handles SSH forwarding with protocol fallback chain",
					"Banner width calculation accounts for CJK and emoji character widths",
					"Daemon socket cleanup on abnormal termination prevents stale socket files",
					"Waifu API timeouts no longer block shell startup when daemon is unavailable",
				},
				Removed: []string{
					"v1 flat configuration format (migration tool provided via `prompt-pulse migrate`)",
					"Legacy single-file architecture (replaced by 29-package modular design)",
					"Hardcoded color values (replaced by theme system)",
					"Direct API calls from shell hooks (replaced by daemon-cached data)",
				},
			},
		},
	}
}

// dcRenderChangelogMarkdown renders a Changelog in Keep a Changelog format.
func dcRenderChangelogMarkdown(cl *Changelog) string {
	var b strings.Builder

	b.WriteString("# Changelog\n\n")
	b.WriteString("All notable changes to prompt-pulse are documented in this file.\n\n")
	b.WriteString("The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),\n")
	b.WriteString("and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).\n\n")

	for _, r := range cl.Releases {
		b.WriteString(fmt.Sprintf("## [%s] - %s\n\n", r.Version, r.Date))

		if len(r.Added) > 0 {
			b.WriteString("### Added\n\n")
			for _, item := range r.Added {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
		}

		if len(r.Changed) > 0 {
			b.WriteString("### Changed\n\n")
			for _, item := range r.Changed {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
		}

		if len(r.Fixed) > 0 {
			b.WriteString("### Fixed\n\n")
			for _, item := range r.Fixed {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
		}

		if len(r.Removed) > 0 {
			b.WriteString("### Removed\n\n")
			for _, item := range r.Removed {
				b.WriteString(fmt.Sprintf("- %s\n", item))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
