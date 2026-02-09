package layout

import (
	"testing"
)

// area is a test helper that creates a Rect at origin with the given size.
func area(w, h int) Rect {
	return Rect{X: 0, Y: 0, Width: w, Height: h}
}

// assertRectsEqual fails the test if got and want differ.
func assertRectsEqual(t *testing.T, label string, got, want []Rect) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len(got)=%d, want %d\ngot:  %v\nwant: %v", label, len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %v, want %v", label, i, got[i], want[i])
		}
	}
}

// --- Fill constraints ---

func TestSingleFillFillsEntireArea(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}).Split(area(100, 50))
	assertRectsEqual(t, "single fill", rects, []Rect{
		{X: 0, Y: 0, Width: 100, Height: 50},
	})
}

func TestTwoFillOneEqualSplit(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Fill{1}).Split(area(100, 50))
	assertRectsEqual(t, "two fills", rects, []Rect{
		{X: 0, Y: 0, Width: 50, Height: 50},
		{X: 50, Y: 0, Width: 50, Height: 50},
	})
}

func TestFillWeightedRatio(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{2}, Fill{1}).Split(area(90, 30))
	// 2:1 ratio => 60, 30
	assertRectsEqual(t, "fill 2:1", rects, []Rect{
		{X: 0, Y: 0, Width: 60, Height: 30},
		{X: 60, Y: 0, Width: 30, Height: 30},
	})
}

func TestFillZeroWeightTreatedAsOne(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{0}, Fill{0}).Split(area(80, 20))
	// Both weight=0 treated as 1, so 40/40
	assertRectsEqual(t, "fill zero weight", rects, []Rect{
		{X: 0, Y: 0, Width: 40, Height: 20},
		{X: 40, Y: 0, Width: 40, Height: 20},
	})
}

// --- Length constraints ---

func TestLengthPlusFill(t *testing.T) {
	rects := NewLayout(Horizontal, Length{10}, Fill{1}).Split(area(100, 50))
	assertRectsEqual(t, "length+fill", rects, []Rect{
		{X: 0, Y: 0, Width: 10, Height: 50},
		{X: 10, Y: 0, Width: 90, Height: 50},
	})
}

func TestMultipleLengths(t *testing.T) {
	rects := NewLayout(Horizontal, Length{20}, Length{30}).Split(area(100, 50))
	// Two fixed lengths, no fill, surplus of 50 left at end (FlexStart default).
	assertRectsEqual(t, "two lengths", rects, []Rect{
		{X: 0, Y: 0, Width: 20, Height: 50},
		{X: 20, Y: 0, Width: 30, Height: 50},
	})
}

// --- Percentage constraints ---

func TestPercentageSumsTo100(t *testing.T) {
	rects := NewLayout(Horizontal, Percentage{30}, Percentage{70}).Split(area(100, 50))
	assertRectsEqual(t, "pct 30+70", rects, []Rect{
		{X: 0, Y: 0, Width: 30, Height: 50},
		{X: 30, Y: 0, Width: 70, Height: 50},
	})
}

func TestPercentageWithOddWidth(t *testing.T) {
	rects := NewLayout(Horizontal, Percentage{50}, Percentage{50}).Split(area(101, 50))
	// 50% of 101 = 50 each (integer truncation), surplus of 1
	if rects[0].Width+rects[1].Width > 101 {
		t.Errorf("percentages exceed available width: %d + %d > 101", rects[0].Width, rects[1].Width)
	}
}

func TestPercentageClampedTo100(t *testing.T) {
	rects := NewLayout(Horizontal, Percentage{150}).Split(area(100, 50))
	// 150 clamped to 100 => 100
	if rects[0].Width != 100 {
		t.Errorf("got width %d, want 100 (percentage clamped to 100%%)", rects[0].Width)
	}
}

func TestPercentageNegativeClamped(t *testing.T) {
	rects := NewLayout(Horizontal, Percentage{-10}).Split(area(100, 50))
	if rects[0].Width != 0 {
		t.Errorf("got width %d, want 0 (negative percentage)", rects[0].Width)
	}
}

// --- Min constraints ---

func TestMinRespectedWhenTight(t *testing.T) {
	// Area = 50 wide. Min(20) + Min(20) = 40 fixed, 10 surplus.
	rects := NewLayout(Horizontal, Min{20}, Min{20}).Split(area(50, 10))
	if rects[0].Width < 20 {
		t.Errorf("Min(20) violated: got width %d", rects[0].Width)
	}
	if rects[1].Width < 20 {
		t.Errorf("Min(20) violated: got width %d", rects[1].Width)
	}
}

func TestMinGrowsToFillSurplus(t *testing.T) {
	// Min(10) in a 100-wide area should get all 100 (it is the only constraint).
	rects := NewLayout(Horizontal, Min{10}).Split(area(100, 50))
	if rects[0].Width < 10 {
		t.Errorf("Min(10) violated: got width %d", rects[0].Width)
	}
}

// --- Max constraints ---

func TestMaxCapsAllocation(t *testing.T) {
	// Max(50) in a 100-wide area: should be capped at 50.
	rects := NewLayout(Horizontal, Max{50}).Split(area(100, 50))
	if rects[0].Width > 50 {
		t.Errorf("Max(50) violated: got width %d", rects[0].Width)
	}
}

func TestMaxWithFill(t *testing.T) {
	// Max(30) + Fill(1) in 100 wide:
	// Max gets capped at 30, Fill gets remainder.
	rects := NewLayout(Horizontal, Max{30}, Fill{1}).Split(area(100, 50))
	if rects[0].Width > 30 {
		t.Errorf("Max(30) violated: got width %d", rects[0].Width)
	}
	if rects[0].Width+rects[1].Width != 100 {
		t.Errorf("total should be 100: got %d + %d = %d",
			rects[0].Width, rects[1].Width, rects[0].Width+rects[1].Width)
	}
}

// --- Ratio constraints ---

func TestRatioOneThirdTwoThirds(t *testing.T) {
	rects := NewLayout(Horizontal, Ratio{1, 3}, Ratio{2, 3}).Split(area(90, 30))
	if rects[0].Width != 30 {
		t.Errorf("Ratio(1,3) of 90: got %d, want 30", rects[0].Width)
	}
	if rects[1].Width != 60 {
		t.Errorf("Ratio(2,3) of 90: got %d, want 60", rects[1].Width)
	}
}

func TestRatioZeroDenominator(t *testing.T) {
	rects := NewLayout(Horizontal, Ratio{1, 0}).Split(area(100, 50))
	if rects[0].Width != 0 {
		t.Errorf("Ratio(1,0) should produce 0, got %d", rects[0].Width)
	}
}

// --- Spacing ---

func TestSpacingBetweenItems(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Fill{1}).
		WithSpacing(4).Split(area(104, 50))
	// 104 - 4 spacing = 100 available. 50/50.
	// First at X=0, second at X=50+4=54
	assertRectsEqual(t, "spacing", rects, []Rect{
		{X: 0, Y: 0, Width: 50, Height: 50},
		{X: 54, Y: 0, Width: 50, Height: 50},
	})
}

func TestSpacingVertical(t *testing.T) {
	rects := NewLayout(Vertical, Fill{1}, Fill{1}).
		WithSpacing(2).Split(area(80, 42))
	// 42 - 2 = 40 available. 20/20.
	assertRectsEqual(t, "vertical spacing", rects, []Rect{
		{X: 0, Y: 0, Width: 80, Height: 20},
		{X: 0, Y: 22, Width: 80, Height: 20},
	})
}

// --- Margin ---

func TestMarginShrinksBothDirections(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}).
		WithMargin(5).Split(area(100, 50))
	// Inner area: X=5, Y=5, W=90, H=40
	assertRectsEqual(t, "margin", rects, []Rect{
		{X: 5, Y: 5, Width: 90, Height: 40},
	})
}

func TestLargeMarginProducesEmptyRects(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}).
		WithMargin(60).Split(area(100, 50))
	// Margin 60 on each side: W=100-120<0 => empty
	if rects[0].Width != 0 || rects[0].Height != 0 {
		t.Errorf("large margin should produce empty rect, got %v", rects[0])
	}
}

// --- Direction ---

func TestVerticalDirection(t *testing.T) {
	rects := NewLayout(Vertical, Length{10}, Fill{1}).Split(area(80, 50))
	assertRectsEqual(t, "vertical", rects, []Rect{
		{X: 0, Y: 0, Width: 80, Height: 10},
		{X: 0, Y: 10, Width: 80, Height: 40},
	})
}

func TestVerticalPercentage(t *testing.T) {
	rects := NewLayout(Vertical, Percentage{25}, Percentage{75}).Split(area(80, 100))
	assertRectsEqual(t, "vertical pct", rects, []Rect{
		{X: 0, Y: 0, Width: 80, Height: 25},
		{X: 0, Y: 25, Width: 80, Height: 75},
	})
}

// --- Zero and edge cases ---

func TestZeroSizeArea(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Fill{1}).Split(area(0, 0))
	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	for i, r := range rects {
		if r.Width != 0 || r.Height != 0 {
			t.Errorf("rect[%d] should be empty, got %v", i, r)
		}
	}
}

func TestNoConstraints(t *testing.T) {
	rects := NewLayout(Horizontal).Split(area(100, 50))
	if rects != nil {
		t.Errorf("no constraints should return nil, got %v", rects)
	}
}

func TestNegativeLengthClampedToZero(t *testing.T) {
	rects := NewLayout(Horizontal, Length{-5}, Fill{1}).Split(area(100, 50))
	if rects[0].Width != 0 {
		t.Errorf("negative length should clamp to 0, got %d", rects[0].Width)
	}
	if rects[1].Width != 100 {
		t.Errorf("fill should get all 100, got %d", rects[1].Width)
	}
}

func TestNegativeSpacingClampedToZero(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Fill{1}).
		WithSpacing(-5).Split(area(100, 50))
	total := rects[0].Width + rects[1].Width
	if total != 100 {
		t.Errorf("negative spacing should be 0, total = %d", total)
	}
}

func TestNegativeMarginClampedToZero(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}).
		WithMargin(-10).Split(area(100, 50))
	if rects[0].Width != 100 {
		t.Errorf("negative margin should be 0, got width %d", rects[0].Width)
	}
}

// --- Flex modes ---

func TestFlexStartDefault(t *testing.T) {
	// Length(30) in 100 wide: item starts at 0.
	rects := NewLayout(Horizontal, Length{30}).Split(area(100, 50))
	if rects[0].X != 0 {
		t.Errorf("FlexStart: expected X=0, got %d", rects[0].X)
	}
}

func TestFlexEnd(t *testing.T) {
	rects := NewLayout(Horizontal, Length{30}).
		WithFlex(FlexEnd).Split(area(100, 50))
	// Surplus=70, item starts at 70.
	if rects[0].X != 70 {
		t.Errorf("FlexEnd: expected X=70, got %d", rects[0].X)
	}
}

func TestFlexCenter(t *testing.T) {
	rects := NewLayout(Horizontal, Length{30}).
		WithFlex(FlexCenter).Split(area(100, 50))
	// Surplus=70, center at 35.
	if rects[0].X != 35 {
		t.Errorf("FlexCenter: expected X=35, got %d", rects[0].X)
	}
}

func TestFlexSpaceBetweenSingleItem(t *testing.T) {
	rects := NewLayout(Horizontal, Length{30}).
		WithFlex(FlexSpaceBetween).Split(area(100, 50))
	// Single item: starts at 0.
	if rects[0].X != 0 {
		t.Errorf("FlexSpaceBetween single: expected X=0, got %d", rects[0].X)
	}
}

func TestFlexSpaceBetweenMultiple(t *testing.T) {
	rects := NewLayout(Horizontal, Length{10}, Length{10}, Length{10}).
		WithFlex(FlexSpaceBetween).Split(area(100, 50))
	// 3 items of 10 = 30. Surplus = 70. Gaps = 70/2 = 35 each.
	// Item0 at X=0, Item1 at X=10+35=45, Item2 at X=45+10+35=90.
	if rects[0].X != 0 {
		t.Errorf("SpaceBetween[0]: X=%d, want 0", rects[0].X)
	}
	if rects[1].X != 45 {
		t.Errorf("SpaceBetween[1]: X=%d, want 45", rects[1].X)
	}
	if rects[2].X != 90 {
		t.Errorf("SpaceBetween[2]: X=%d, want 90", rects[2].X)
	}
}

func TestFlexSpaceEvenly(t *testing.T) {
	rects := NewLayout(Horizontal, Length{20}, Length{20}).
		WithFlex(FlexSpaceEvenly).Split(area(100, 50))
	// 2 items = 40. Surplus = 60. Slots = 3. Gap = 20.
	// Item0 at X=20, Item1 at X=20+20+20=60.
	if rects[0].X != 20 {
		t.Errorf("SpaceEvenly[0]: X=%d, want 20", rects[0].X)
	}
	if rects[1].X != 60 {
		t.Errorf("SpaceEvenly[1]: X=%d, want 60", rects[1].X)
	}
}

// --- Combined constraints ---

func TestLengthPercentageFill(t *testing.T) {
	rects := NewLayout(Horizontal, Length{20}, Percentage{30}, Fill{1}).Split(area(100, 50))
	// Length=20, Pct=30, Fill=100-20-30=50
	assertRectsEqual(t, "len+pct+fill", rects, []Rect{
		{X: 0, Y: 0, Width: 20, Height: 50},
		{X: 20, Y: 0, Width: 30, Height: 50},
		{X: 50, Y: 0, Width: 50, Height: 50},
	})
}

func TestThreeFillsWeighted(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Fill{2}, Fill{1}).Split(area(80, 40))
	// 1:2:1 => 20:40:20
	assertRectsEqual(t, "3 fills", rects, []Rect{
		{X: 0, Y: 0, Width: 20, Height: 40},
		{X: 20, Y: 0, Width: 40, Height: 40},
		{X: 60, Y: 0, Width: 20, Height: 40},
	})
}

// --- Non-overlapping guarantee ---

func TestNoOverlap(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Length{20}, Percentage{30}, Fill{2}).Split(area(200, 100))
	for i := 0; i < len(rects); i++ {
		for j := i + 1; j < len(rects); j++ {
			inter := rects[i].Intersect(rects[j])
			if !inter.Empty() {
				t.Errorf("rects[%d] and rects[%d] overlap: %v intersect %v = %v",
					i, j, rects[i], rects[j], inter)
			}
		}
	}
}

func TestNoOverlapVertical(t *testing.T) {
	rects := NewLayout(Vertical, Fill{1}, Length{5}, Percentage{20}, Fill{3}).Split(area(80, 100))
	for i := 0; i < len(rects); i++ {
		for j := i + 1; j < len(rects); j++ {
			inter := rects[i].Intersect(rects[j])
			if !inter.Empty() {
				t.Errorf("rects[%d] and rects[%d] overlap: %v intersect %v = %v",
					i, j, rects[i], rects[j], inter)
			}
		}
	}
}

// --- Rect methods ---

func TestRectArea(t *testing.T) {
	r := Rect{X: 0, Y: 0, Width: 10, Height: 5}
	if r.Area() != 50 {
		t.Errorf("Area: got %d, want 50", r.Area())
	}
}

func TestRectEmpty(t *testing.T) {
	if !(Rect{Width: 0, Height: 5}).Empty() {
		t.Error("zero width should be empty")
	}
	if !(Rect{Width: 5, Height: 0}).Empty() {
		t.Error("zero height should be empty")
	}
	if (Rect{Width: 5, Height: 5}).Empty() {
		t.Error("5x5 should not be empty")
	}
}

func TestRectContains(t *testing.T) {
	r := Rect{X: 10, Y: 20, Width: 30, Height: 40}
	tests := []struct {
		x, y int
		want bool
	}{
		{10, 20, true},  // top-left corner (inclusive)
		{39, 59, true},  // bottom-right (inclusive, last pixel)
		{40, 60, false}, // right/bottom edge (exclusive)
		{9, 20, false},  // left of rect
		{25, 15, false}, // above rect
	}
	for _, tt := range tests {
		got := r.Contains(tt.x, tt.y)
		if got != tt.want {
			t.Errorf("Contains(%d,%d): got %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestRectIntersect(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	b := Rect{X: 5, Y: 5, Width: 10, Height: 10}
	inter := a.Intersect(b)
	want := Rect{X: 5, Y: 5, Width: 5, Height: 5}
	if inter != want {
		t.Errorf("Intersect: got %v, want %v", inter, want)
	}
}

func TestRectIntersectNoOverlap(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 5, Height: 5}
	b := Rect{X: 10, Y: 10, Width: 5, Height: 5}
	inter := a.Intersect(b)
	if !inter.Empty() {
		t.Errorf("non-overlapping rects should produce empty intersect, got %v", inter)
	}
}

func TestRectInner(t *testing.T) {
	r := Rect{X: 10, Y: 10, Width: 100, Height: 50}
	inner := r.Inner(5)
	want := Rect{X: 15, Y: 15, Width: 90, Height: 40}
	if inner != want {
		t.Errorf("Inner(5): got %v, want %v", inner, want)
	}
}

func TestRectInnerNegativeMargin(t *testing.T) {
	r := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	inner := r.Inner(-3)
	// Negative margin clamped to 0.
	if inner != r {
		t.Errorf("Inner(-3) should equal original, got %v", inner)
	}
}

func TestRectInnerLargerThanRect(t *testing.T) {
	r := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	inner := r.Inner(20)
	if inner.Width != 0 || inner.Height != 0 {
		t.Errorf("oversized inner should have zero dimensions, got %v", inner)
	}
}

// --- Helpers ---

func TestSplitVerticalHelper(t *testing.T) {
	rects := SplitVertical(area(80, 100), Fill{1}, Fill{1})
	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	if rects[0].Height != 50 || rects[1].Height != 50 {
		t.Errorf("vertical split: heights %d, %d (want 50, 50)", rects[0].Height, rects[1].Height)
	}
}

func TestSplitHorizontalHelper(t *testing.T) {
	rects := SplitHorizontal(area(100, 80), Fill{1}, Fill{1})
	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	if rects[0].Width != 50 || rects[1].Width != 50 {
		t.Errorf("horizontal split: widths %d, %d (want 50, 50)", rects[0].Width, rects[1].Width)
	}
}

func TestLayoutBuilder(t *testing.T) {
	rects := NewLayoutBuilder().
		Direction(Horizontal).
		Constraints(Length{20}, Fill{1}).
		Spacing(2).
		Margin(5).
		Split(area(100, 50))
	// inner: X=5 Y=5 W=90 H=40
	// available=90-2=88, Length=20, Fill=68
	if len(rects) != 2 {
		t.Fatalf("builder: expected 2 rects, got %d", len(rects))
	}
	if rects[0].Width != 20 {
		t.Errorf("builder: first width=%d, want 20", rects[0].Width)
	}
	if rects[1].Width != 68 {
		t.Errorf("builder: second width=%d, want 68", rects[1].Width)
	}
	if rects[0].X != 5 {
		t.Errorf("builder: first X=%d, want 5 (margin)", rects[0].X)
	}
}

func TestLayoutBuilderBuild(t *testing.T) {
	l := NewLayoutBuilder().
		Direction(Vertical).
		Constraints(Fill{1}, Fill{1}).
		Build()
	rects := l.Split(area(80, 100))
	if len(rects) != 2 {
		t.Fatalf("build: expected 2 rects, got %d", len(rects))
	}
	if rects[0].Height != 50 || rects[1].Height != 50 {
		t.Errorf("build: heights %d, %d (want 50, 50)", rects[0].Height, rects[1].Height)
	}
}

// --- Cache tests ---

func TestCacheHitReturnsSameResult(t *testing.T) {
	cache := NewLayoutCache()
	l := NewLayout(Horizontal, Fill{1}, Fill{1})
	a := area(100, 50)

	r1 := cache.SplitCached(l, a)
	r2 := cache.SplitCached(l, a)

	assertRectsEqual(t, "cache hit", r1, r2)
	if cache.Len() != 1 {
		t.Errorf("cache should have 1 entry, got %d", cache.Len())
	}
}

func TestCacheMissOnDifferentArea(t *testing.T) {
	cache := NewLayoutCache()
	l := NewLayout(Horizontal, Fill{1})

	r1 := cache.SplitCached(l, area(100, 50))
	r2 := cache.SplitCached(l, area(200, 50))

	if r1[0].Width == r2[0].Width {
		t.Error("different areas should produce different widths")
	}
	if cache.Len() != 2 {
		t.Errorf("cache should have 2 entries, got %d", cache.Len())
	}
}

func TestCacheInvalidate(t *testing.T) {
	cache := NewLayoutCache()
	l := NewLayout(Horizontal, Fill{1})
	cache.SplitCached(l, area(100, 50))
	cache.SplitCached(l, area(200, 50))

	cache.Invalidate()
	if cache.Len() != 0 {
		t.Errorf("cache should be empty after invalidate, got %d", cache.Len())
	}
}

func TestCacheReturnsCopy(t *testing.T) {
	cache := NewLayoutCache()
	l := NewLayout(Horizontal, Fill{1})
	a := area(100, 50)

	r1 := cache.SplitCached(l, a)
	r1[0].Width = 999 // mutate the returned slice

	r2 := cache.SplitCached(l, a)
	if r2[0].Width == 999 {
		t.Error("cache should return a copy, but mutation leaked")
	}
}

// --- Over-allocation (stress) ---

func TestOverAllocationShrinks(t *testing.T) {
	// Two Length constraints that exceed available space.
	rects := NewLayout(Horizontal, Length{80}, Length{80}).Split(area(100, 50))
	total := rects[0].Width + rects[1].Width
	if total > 100 {
		t.Errorf("over-allocation should shrink: total=%d > 100", total)
	}
}

// --- Real-world: TUI banner layout ---

func TestTUIBannerLayout(t *testing.T) {
	// Simulates the prompt-pulse banner: image | info | sparklines
	term := area(120, 40)
	cols := NewLayout(Horizontal, Length{28}, Fill{1}, Length{20}).Split(term)
	if cols[0].Width != 28 {
		t.Errorf("image col: got %d, want 28", cols[0].Width)
	}
	if cols[2].Width != 20 {
		t.Errorf("sparkline col: got %d, want 20", cols[2].Width)
	}
	infoWidth := 120 - 28 - 20
	if cols[1].Width != infoWidth {
		t.Errorf("info col: got %d, want %d", cols[1].Width, infoWidth)
	}
}

func TestTUINestedLayout(t *testing.T) {
	// Top-level: header + body
	outer := NewLayout(Vertical, Length{3}, Fill{1}).Split(area(120, 40))
	if outer[0].Height != 3 {
		t.Errorf("header height: got %d, want 3", outer[0].Height)
	}

	// Body split into 3 columns.
	body := outer[1]
	cols := NewLayout(Horizontal, Length{28}, Fill{1}, Length{20}).Split(body)
	if cols[0].Width != 28 {
		t.Errorf("nested image col: got %d, want 28", cols[0].Width)
	}
	if cols[0].Y != 3 {
		t.Errorf("nested Y offset: got %d, want 3", cols[0].Y)
	}
}

// --- Offset area ---

func TestOffsetArea(t *testing.T) {
	a := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	rects := NewLayout(Horizontal, Fill{1}, Fill{1}).Split(a)
	if rects[0].X != 10 {
		t.Errorf("first rect X: got %d, want 10", rects[0].X)
	}
	if rects[1].X != 60 {
		t.Errorf("second rect X: got %d, want 60", rects[1].X)
	}
	if rects[0].Y != 20 || rects[1].Y != 20 {
		t.Errorf("Y should be preserved: got %d, %d", rects[0].Y, rects[1].Y)
	}
}

// --- Flex with spacing ---

func TestFlexEndWithSpacing(t *testing.T) {
	rects := NewLayout(Horizontal, Length{10}, Length{10}).
		WithFlex(FlexEnd).
		WithSpacing(5).Split(area(100, 50))
	// available = 100 - 5 = 95, allocated = 20, surplus = 75
	// FlexEnd: offset by surplus=75
	if rects[0].X != 75 {
		t.Errorf("FlexEnd+spacing [0].X: got %d, want 75", rects[0].X)
	}
	if rects[1].X != 90 {
		t.Errorf("FlexEnd+spacing [1].X: got %d, want 90 (75+10+5)", rects[1].X)
	}
}

// --- Many items ---

func TestManyFills(t *testing.T) {
	n := 10
	cs := make([]Constraint, n)
	for i := range cs {
		cs[i] = Fill{1}
	}
	rects := NewLayout(Horizontal, cs...).Split(area(100, 50))
	if len(rects) != n {
		t.Fatalf("expected %d rects, got %d", n, len(rects))
	}
	total := 0
	for _, r := range rects {
		total += r.Width
	}
	if total != 100 {
		t.Errorf("total width should be 100, got %d", total)
	}
}

// --- Spacing exceeds available space ---

func TestSpacingExceedsSpace(t *testing.T) {
	rects := NewLayout(Horizontal, Fill{1}, Fill{1}).
		WithSpacing(200).Split(area(100, 50))
	// spacing = 200 > 100 available, so available after spacing = 0
	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	// Both should have width 0.
	for i, r := range rects {
		if r.Width != 0 {
			t.Errorf("rect[%d].Width should be 0, got %d", i, r.Width)
		}
	}
}
