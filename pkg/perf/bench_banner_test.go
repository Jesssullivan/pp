package perf

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/banner"
)

// pfMakeBannerData builds a realistic BannerData with 6 populated WidgetData
// entries resembling a typical dashboard layout: system metrics, Claude usage,
// billing, Tailscale, Kubernetes status, and a waifu image placeholder.
func pfMakeBannerData() banner.BannerData {
	widgets := []banner.WidgetData{
		{
			ID:    "sysmetrics",
			Title: "System",
			Content: strings.Join([]string{
				"CPU  \x1b[38;2;76;175;80m████████████\x1b[0m\x1b[38;2;51;51;51m        \x1b[0m 62%",
				"RAM  \x1b[38;2;76;175;80m██████████████\x1b[0m\x1b[38;2;51;51;51m      \x1b[0m 73%",
				"Disk \x1b[38;2;76;175;80m████████████████\x1b[0m\x1b[38;2;51;51;51m    \x1b[0m 81%",
				"Net  \x1b[38;2;76;175;80m██████\x1b[0m\x1b[38;2;51;51;51m              \x1b[0m 31%",
			}, "\n"),
			MinW: 30,
			MinH: 6,
		},
		{
			ID:    "claude",
			Title: "Claude Usage",
			Content: strings.Join([]string{
				"Month:    $142.30",
				"Model:    claude-opus-4-6",
				"Tokens:   1.2M input / 384K output",
				"Sessions: 47 active",
			}, "\n"),
			MinW: 35,
			MinH: 6,
		},
		{
			ID:    "billing",
			Title: "Cloud Billing",
			Content: strings.Join([]string{
				"DigitalOcean  $23.45/mo",
				"GitLab CI     $12.80/mo",
				"Total         $36.25/mo",
			}, "\n"),
			MinW: 30,
			MinH: 5,
		},
		{
			ID:    "tailscale",
			Title: "Tailscale",
			Content: strings.Join([]string{
				"Status: Connected",
				"Peers:  5/7 online",
				"IP:     100.64.0.1",
			}, "\n"),
			MinW: 25,
			MinH: 5,
		},
		{
			ID:    "k8s",
			Title: "Kubernetes",
			Content: strings.Join([]string{
				"Cluster: tinyland-prod",
				"Pods:    12/15 Running",
				"Nodes:   3/3 Ready",
				"Failed:  0",
			}, "\n"),
			MinW: 30,
			MinH: 6,
		},
		{
			ID:      "waifu-main",
			Title:   "Waifu",
			Content: strings.Repeat("W", 40*20),
			MinW:    40,
			MinH:    20,
		},
	}

	return banner.BannerData{Widgets: widgets}
}

// BenchmarkBannerRenderCompact benchmarks Render at the Compact preset (80x24)
// with 6 realistic widgets.
func BenchmarkBannerRenderCompact(b *testing.B) {
	data := pfMakeBannerData()
	preset := banner.Compact

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = banner.Render(data, preset)
	}
}

// BenchmarkBannerRenderStandard benchmarks Render at the Standard preset
// (120x35) with 6 realistic widgets.
func BenchmarkBannerRenderStandard(b *testing.B) {
	data := pfMakeBannerData()
	preset := banner.Standard

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = banner.Render(data, preset)
	}
}

// BenchmarkBannerRenderCached benchmarks the cached banner read path. A cache
// is pre-warmed so every iteration hits the fast path (stat + read, no render).
func BenchmarkBannerRenderCached(b *testing.B) {
	data := pfMakeBannerData()
	preset := banner.Standard

	cacheDir, err := os.MkdirTemp("", "bench-banner-cache-*")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(cacheDir)

	// Warm the cache.
	_, err = banner.RenderCached(cacheDir, data, preset)
	if err != nil {
		b.Fatalf("warm cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = banner.RenderCached(cacheDir, data, preset)
	}
}

// BenchmarkBannerSelectPreset benchmarks the preset selection logic across
// a range of terminal sizes.
func BenchmarkBannerSelectPreset(b *testing.B) {
	// Cycle through various terminal sizes to exercise all branches.
	sizes := [][2]int{
		{60, 20},   // too small for compact
		{80, 24},   // compact
		{120, 35},  // standard
		{160, 45},  // wide
		{200, 50},  // ultrawide
		{100, 30},  // between compact and standard
		{250, 60},  // beyond ultrawide
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := sizes[i%len(sizes)]
		_ = banner.SelectPreset(s[0], s[1])
	}
}

// pfBannerRenderForTest is a helper used by tests to verify benchmarks run
// without panic by calling them once.
func pfBannerRenderForTest() string {
	data := pfMakeBannerData()
	return fmt.Sprintf("widgets=%d", len(data.Widgets))
}
