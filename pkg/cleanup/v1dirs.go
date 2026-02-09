package cleanup

// V1Directory describes a known v1 directory, its purpose, and its v2 replacement.
type V1Directory struct {
	Path          string
	Description   string
	V2Replacement string
	Safe          bool
}

// clV1Directories returns the full list of known v1 directories with their
// corresponding v2 replacements. These mappings are hand-curated based on the
// actual migration that took place over weeks 1-9.
func clV1Directories() []V1Directory {
	return []V1Directory{
		{
			Path:          "display/banner",
			Description:   "v1 banner rendering: layout, box drawing, terminal detection",
			V2Replacement: "pkg/layout/, pkg/terminal/, pkg/components/",
			Safe:          true,
		},
		{
			Path:          "display/color",
			Description:   "v1 color and theme system",
			V2Replacement: "pkg/theme/",
			Safe:          true,
		},
		{
			Path:          "display/layout",
			Description:   "v1 responsive layout (grid-based)",
			V2Replacement: "pkg/layout/ (Cassowary constraint solver)",
			Safe:          true,
		},
		{
			Path:          "display/render",
			Description:   "v1 image rendering (chafa, iterm2, sixel)",
			V2Replacement: "pkg/image/ (multi-protocol renderer)",
			Safe:          true,
		},
		{
			Path:          "display/starship",
			Description:   "v1 Starship config and output parsing",
			V2Replacement: "pkg/starship/",
			Safe:          true,
		},
		{
			Path:          "display/tui",
			Description:   "v1 TUI application (Bubble Tea app, tabs, keys, theme)",
			V2Replacement: "pkg/tui/ + pkg/widgets/",
			Safe:          true,
		},
		{
			Path:          "display/widgets",
			Description:   "v1 widget components (gauge, sparkline, table, panels)",
			V2Replacement: "pkg/widgets/",
			Safe:          true,
		},
		{
			Path:          "waifu",
			Description:   "v1 waifu image rendering and session management",
			V2Replacement: "pkg/waifu/ + pkg/image/",
			Safe:          true,
		},
		{
			Path:          "collectors",
			Description:   "v1 data collectors (billing, claude, infra, sysmetrics)",
			V2Replacement: "pkg/collectors/ (duck-typed interface)",
			Safe:          true,
		},
		{
			Path:          "shell",
			Description:   "v1 shell integration (bash, zsh, fish, nushell)",
			V2Replacement: "pkg/shell/ (4-shell support)",
			Safe:          true,
		},
		{
			Path:          "config",
			Description:   "v1 configuration system",
			V2Replacement: "pkg/config/ (nested TOML)",
			Safe:          true,
		},
		{
			Path:          "cache",
			Description:   "v1 cache store",
			V2Replacement: "pkg/cache/ (LRU + atomic writes)",
			Safe:          true,
		},
		{
			Path:          "status",
			Description:   "v1 status evaluator and selector",
			V2Replacement: "pkg/widgets/ (sysmetrics widget)",
			Safe:          true,
		},
		{
			Path:          "internal/format",
			Description:   "v1 internal formatting utilities (strings, time)",
			V2Replacement: "various pkg/ packages",
			Safe:          true,
		},
		{
			Path:          "cmd/demo-mocks",
			Description:   "v1 CLI demo-mocks subcommand",
			V2Replacement: "pkg/tui/ + pkg/banner/",
			Safe:          true,
		},
		{
			Path:          "scripts",
			Description:   "v1 shell scripts (check-secrets.sh)",
			V2Replacement: "Go-native implementations",
			Safe:          false,
		},
	}
}
