package perf

import (
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/shell"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/starship"
)

// pfMakeShellOptions returns a realistic shell.Options configuration with all
// features enabled for benchmarking the maximum generation path.
func pfMakeShellOptions() shell.Options {
	return shell.Options{
		BinaryPath:        "/usr/local/bin/prompt-pulse",
		Keybinding:        `\C-p`,
		ShowBanner:        true,
		DaemonAutoStart:   true,
		EnableCompletions: true,
	}
}

// BenchmarkShellGenerateBash benchmarks generating the full Bash integration
// script with all features enabled (banner, keybinding, completions, daemon).
func BenchmarkShellGenerateBash(b *testing.B) {
	opts := pfMakeShellOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shell.Generate(shell.Bash, opts)
	}
}

// BenchmarkShellGenerateZsh benchmarks generating the full Zsh integration
// script with all features enabled (banner, ZLE widget, completions, daemon).
func BenchmarkShellGenerateZsh(b *testing.B) {
	opts := pfMakeShellOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shell.Generate(shell.Zsh, opts)
	}
}

// BenchmarkStarshipRender benchmarks the starship segment rendering with all
// segment types enabled. Note: this will produce empty output in CI/test
// environments where no cached collector data exists, but it still exercises
// the full rendering pipeline including cache lookups and formatting.
func BenchmarkStarshipRender(b *testing.B) {
	cfg := starship.Config{
		ShowClaude:    true,
		ShowBilling:   true,
		ShowTailscale: true,
		ShowK8s:       true,
		ShowSystem:    true,
		CacheDir:      b.TempDir(),
		MaxWidth:      60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = starship.Render(cfg)
	}
}
