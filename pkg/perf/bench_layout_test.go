package perf

import (
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/layout"
)

// pfMakeLayoutConstraints6 returns a realistic set of 6 constraints for a
// dashboard layout: two percentage columns, two fill columns, one min, one
// fixed-width column.
func pfMakeLayoutConstraints6() []layout.Constraint {
	return []layout.Constraint{
		layout.Percentage{Value: 20},
		layout.Fill{Weight: 2},
		layout.Min{Value: 10},
		layout.Percentage{Value: 15},
		layout.Fill{Weight: 1},
		layout.Length{Value: 8},
	}
}

// pfMakeLayoutConstraints20 returns a stress-test set of 20 constraints mixing
// all constraint types.
func pfMakeLayoutConstraints20() []layout.Constraint {
	cs := make([]layout.Constraint, 20)
	for i := 0; i < 20; i++ {
		switch i % 6 {
		case 0:
			cs[i] = layout.Percentage{Value: 10}
		case 1:
			cs[i] = layout.Fill{Weight: 1}
		case 2:
			cs[i] = layout.Min{Value: 5}
		case 3:
			cs[i] = layout.Max{Value: 20}
		case 4:
			cs[i] = layout.Length{Value: 6}
		case 5:
			cs[i] = layout.Ratio{Num: 1, Den: 20}
		}
	}
	return cs
}

// BenchmarkLayoutSolve6Widgets benchmarks the layout solver with 6 constraints
// in a horizontal split, simulating a typical dashboard column layout.
func BenchmarkLayoutSolve6Widgets(b *testing.B) {
	constraints := pfMakeLayoutConstraints6()
	area := layout.Rect{X: 0, Y: 0, Width: 120, Height: 35}
	l := layout.NewLayout(layout.Horizontal, constraints...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Split(area)
	}
}

// BenchmarkLayoutSolve20Widgets benchmarks the layout solver under stress with
// 20 constraints in a vertical split.
func BenchmarkLayoutSolve20Widgets(b *testing.B) {
	constraints := pfMakeLayoutConstraints20()
	area := layout.Rect{X: 0, Y: 0, Width: 200, Height: 60}
	l := layout.NewLayout(layout.Vertical, constraints...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Split(area)
	}
}

// BenchmarkLayoutCacheHit benchmarks the cached layout lookup path. The cache
// is pre-warmed so every iteration is a cache hit.
func BenchmarkLayoutCacheHit(b *testing.B) {
	constraints := pfMakeLayoutConstraints6()
	area := layout.Rect{X: 0, Y: 0, Width: 120, Height: 35}
	l := layout.NewLayout(layout.Horizontal, constraints...)

	cache := layout.NewLayoutCache()
	// Warm the cache.
	cache.SplitCached(l, area)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.SplitCached(l, area)
	}
}
