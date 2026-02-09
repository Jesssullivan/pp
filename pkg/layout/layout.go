// Package layout provides a Cassowary-inspired constraint-based layout engine
// for terminal user interfaces. It splits rectangular areas into sub-regions
// according to declarative constraints, similar to ratatui's layout system.
//
// Constraint types:
//   - Length(n): fixed size in cells
//   - Percentage(p): percentage of available space (0-100)
//   - Min(n): at least n cells, grows to fill surplus
//   - Max(n): at most n cells
//   - Fill(w): fills remaining space proportional to weight
//   - Ratio(n,d): n/d of total available space
//
// The solver runs in three passes:
//  1. Allocate fixed sizes (Length, Percentage, Ratio)
//  2. Distribute remaining space to Fill items by weight
//  3. Enforce Min/Max bounds, redistribute overflow
//
// Any leftover surplus is positioned according to the Flex mode.
package layout

// Rect represents a rectangular area in terminal cells.
type Rect struct {
	X, Y, Width, Height int
}

// Area returns the number of cells in this rectangle.
func (r Rect) Area() int {
	return r.Width * r.Height
}

// Empty returns true if this rectangle has zero area.
func (r Rect) Empty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Right returns the X coordinate of the right edge (exclusive).
func (r Rect) Right() int {
	return r.X + r.Width
}

// Bottom returns the Y coordinate of the bottom edge (exclusive).
func (r Rect) Bottom() int {
	return r.Y + r.Height
}

// Inner returns a new Rect shrunk by margin on all sides.
// If the margin would cause negative dimensions, a zero-size rect is returned.
func (r Rect) Inner(margin int) Rect {
	if margin < 0 {
		margin = 0
	}
	x := r.X + margin
	y := r.Y + margin
	w := r.Width - 2*margin
	h := r.Height - 2*margin
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x, Y: y, Width: w, Height: h}
}

// Contains returns true if the point (px, py) lies within this rectangle.
func (r Rect) Contains(px, py int) bool {
	return px >= r.X && px < r.Right() && py >= r.Y && py < r.Bottom()
}

// Intersect returns the overlapping region of two rectangles.
// If there is no overlap, returns a zero-size Rect.
func (r Rect) Intersect(other Rect) Rect {
	x1 := maxInt(r.X, other.X)
	y1 := maxInt(r.Y, other.Y)
	x2 := minInt(r.Right(), other.Right())
	y2 := minInt(r.Bottom(), other.Bottom())
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// Direction controls the axis along which a Layout splits space.
type Direction int

const (
	// Horizontal splits left-to-right (constraints control width).
	Horizontal Direction = iota
	// Vertical splits top-to-bottom (constraints control height).
	Vertical
)

// Constraint is the interface satisfied by all layout constraint types.
// The marker method prevents external implementations.
type Constraint interface {
	constraint() // sealed marker
}

// Length allocates exactly Value cells.
type Length struct{ Value int }

func (Length) constraint() {}

// Percentage allocates Value percent of the total available space (0-100).
type Percentage struct{ Value int }

func (Percentage) constraint() {}

// Min allocates at least Value cells. If surplus exists after fixed
// allocations, Min items grow proportionally to fill it.
type Min struct{ Value int }

func (Min) constraint() {}

// Max allocates at most Value cells.
type Max struct{ Value int }

func (Max) constraint() {}

// Fill distributes remaining space proportional to Weight.
// A Weight of 0 is treated as 1.
type Fill struct{ Weight int }

func (Fill) constraint() {}

// Ratio allocates Num/Den of the total available space.
type Ratio struct{ Num, Den int }

func (Ratio) constraint() {}

// Flex controls how surplus space (space remaining after all constraints
// are satisfied) is distributed.
type Flex int

const (
	// FlexStart packs items to the start; surplus goes at the end.
	FlexStart Flex = iota
	// FlexEnd packs items to the end; surplus goes at the start.
	FlexEnd
	// FlexCenter centers items; surplus is split equally on both sides.
	FlexCenter
	// FlexSpaceBetween distributes surplus evenly between items.
	FlexSpaceBetween
	// FlexSpaceAround distributes surplus evenly around items (half-gaps at edges).
	FlexSpaceAround
	// FlexSpaceEvenly distributes surplus evenly in all gaps (including edges).
	FlexSpaceEvenly
)

// Layout splits a Rect into sub-regions according to constraints.
type Layout struct {
	direction   Direction
	constraints []Constraint
	flex        Flex
	spacing     int
	margin      int
}

// NewLayout creates a Layout with the given direction and constraints.
func NewLayout(dir Direction, constraints ...Constraint) *Layout {
	return &Layout{
		direction:   dir,
		constraints: constraints,
	}
}

// WithFlex sets the flex mode for surplus distribution.
func (l *Layout) WithFlex(f Flex) *Layout {
	l.flex = f
	return l
}

// WithSpacing sets the gap (in cells) between each output region.
func (l *Layout) WithSpacing(s int) *Layout {
	if s < 0 {
		s = 0
	}
	l.spacing = s
	return l
}

// WithMargin sets the outer margin (in cells) on all sides of the input area.
func (l *Layout) WithMargin(m int) *Layout {
	if m < 0 {
		m = 0
	}
	l.margin = m
	return l
}

// Split divides area into len(constraints) non-overlapping Rects.
//
// Algorithm:
//  1. Apply margin to shrink the working area.
//  2. Subtract total spacing from available space.
//  3. First pass: compute raw allocation per constraint.
//  4. Second pass: distribute remaining space to Fill/Min items.
//  5. Third pass: enforce Min/Max, redistribute overflow.
//  6. Clamp all allocations to non-negative.
//  7. Apply Flex distribution to any surplus.
//  8. Convert 1-D allocations to Rects along the layout direction.
func (l *Layout) Split(area Rect) []Rect {
	n := len(l.constraints)
	if n == 0 {
		return nil
	}

	// Apply margin.
	inner := area.Inner(l.margin)
	if inner.Empty() {
		return makeEmptyRects(n, inner)
	}

	// Total available space along the layout axis.
	total := l.axisSize(inner)

	// Subtract spacing between items.
	spacingTotal := 0
	if n > 1 {
		spacingTotal = l.spacing * (n - 1)
	}
	available := total - spacingTotal
	if available < 0 {
		available = 0
	}

	// --- Pass 1: raw allocations ---
	allocs := make([]int, n)
	isFill := make([]bool, n)   // tracks Fill constraints
	isMin := make([]bool, n)    // tracks Min constraints
	isMax := make([]bool, n)    // tracks Max constraints
	fillWeights := make([]int, n)
	minValues := make([]int, n)
	maxValues := make([]int, n)

	totalFillWeight := 0
	fixedUsed := 0

	for i, c := range l.constraints {
		switch v := c.(type) {
		case Length:
			allocs[i] = clampNonNeg(v.Value)
			fixedUsed += allocs[i]
		case Percentage:
			p := clampRange(v.Value, 0, 100)
			allocs[i] = available * p / 100
			fixedUsed += allocs[i]
		case Ratio:
			if v.Den <= 0 {
				allocs[i] = 0
			} else {
				allocs[i] = available * clampNonNeg(v.Num) / v.Den
			}
			fixedUsed += allocs[i]
		case Fill:
			w := v.Weight
			if w <= 0 {
				w = 1
			}
			isFill[i] = true
			fillWeights[i] = w
			totalFillWeight += w
			allocs[i] = 0 // filled in pass 2
		case Min:
			isMin[i] = true
			minValues[i] = clampNonNeg(v.Value)
			allocs[i] = minValues[i]
			fixedUsed += allocs[i]
		case Max:
			isMax[i] = true
			maxValues[i] = clampNonNeg(v.Value)
			// Max items start at the full remaining space (capped later).
			// For pass 1, treat as fill-like but we will cap in pass 3.
			isFill[i] = true
			fillWeights[i] = 1
			totalFillWeight += 1
			allocs[i] = 0
		}
	}

	// --- Pass 2: distribute remaining space to Fill (and Max-as-fill) items ---
	remaining := available - fixedUsed
	if remaining < 0 {
		remaining = 0
	}

	if totalFillWeight > 0 && remaining > 0 {
		distributed := 0
		lastFill := -1
		for i := 0; i < n; i++ {
			if isFill[i] {
				lastFill = i
			}
		}
		for i := 0; i < n; i++ {
			if !isFill[i] {
				continue
			}
			if i == lastFill {
				// Last fill gets the remainder to avoid rounding drift.
				allocs[i] = remaining - distributed
			} else {
				allocs[i] = remaining * fillWeights[i] / totalFillWeight
				distributed += allocs[i]
			}
		}
	}

	// --- Pass 3: enforce Min/Max constraints, redistribute overflow ---
	// Run up to n iterations to converge (each iteration can free space for others).
	for iter := 0; iter < n; iter++ {
		changed := false
		overflow := 0

		for i := 0; i < n; i++ {
			if isMin[i] && allocs[i] < minValues[i] {
				overflow += minValues[i] - allocs[i]
				allocs[i] = minValues[i]
				changed = true
			}
			if isMax[i] && allocs[i] > maxValues[i] {
				overflow -= allocs[i] - maxValues[i]
				allocs[i] = maxValues[i]
				changed = true
			}
		}

		// Redistribute overflow among non-constrained items.
		if overflow != 0 {
			for i := 0; i < n; i++ {
				if isMin[i] || isMax[i] {
					continue
				}
				if isFill[i] || (!isMin[i] && !isMax[i]) {
					allocs[i] -= overflow
					if allocs[i] < 0 {
						allocs[i] = 0
					}
					break
				}
			}
		}

		if !changed {
			break
		}
	}

	// Clamp all to non-negative.
	for i := range allocs {
		if allocs[i] < 0 {
			allocs[i] = 0
		}
	}

	// Calculate total allocated and surplus.
	totalAllocated := 0
	for _, a := range allocs {
		totalAllocated += a
	}
	surplus := available - totalAllocated
	if surplus < 0 {
		// Over-allocated: proportionally shrink to fit.
		l.shrinkToFit(allocs, available)
		surplus = 0
	}

	// --- Flex distribution ---
	offsets := l.computeOffsets(allocs, surplus, n)

	// --- Build output Rects ---
	rects := make([]Rect, n)
	for i := 0; i < n; i++ {
		switch l.direction {
		case Horizontal:
			rects[i] = Rect{
				X:      inner.X + offsets[i],
				Y:      inner.Y,
				Width:  allocs[i],
				Height: inner.Height,
			}
		case Vertical:
			rects[i] = Rect{
				X:      inner.X,
				Y:      inner.Y + offsets[i],
				Width:  inner.Width,
				Height: allocs[i],
			}
		}
	}

	return rects
}

// computeOffsets converts allocations into start positions, applying
// spacing and flex distribution.
func (l *Layout) computeOffsets(allocs []int, surplus int, n int) []int {
	offsets := make([]int, n)
	if n == 0 {
		return offsets
	}

	switch l.flex {
	case FlexStart:
		pos := 0
		for i := 0; i < n; i++ {
			offsets[i] = pos
			pos += allocs[i] + l.spacing
		}

	case FlexEnd:
		pos := surplus
		for i := 0; i < n; i++ {
			offsets[i] = pos
			pos += allocs[i] + l.spacing
		}

	case FlexCenter:
		pos := surplus / 2
		for i := 0; i < n; i++ {
			offsets[i] = pos
			pos += allocs[i] + l.spacing
		}

	case FlexSpaceBetween:
		if n == 1 {
			offsets[0] = 0
		} else {
			gap := 0
			if n > 1 {
				gap = surplus / (n - 1)
			}
			extra := 0
			if n > 1 {
				extra = surplus % (n - 1)
			}
			pos := 0
			for i := 0; i < n; i++ {
				offsets[i] = pos
				g := gap
				if i < extra {
					g++
				}
				pos += allocs[i] + l.spacing + g
			}
		}

	case FlexSpaceAround:
		gap := 0
		if n > 0 {
			gap = surplus / (2 * n)
		}
		pos := gap
		for i := 0; i < n; i++ {
			offsets[i] = pos
			pos += allocs[i] + l.spacing + 2*gap
		}

	case FlexSpaceEvenly:
		slots := n + 1
		gap := surplus / slots
		extra := surplus % slots
		pos := gap
		if extra > 0 {
			pos++
			extra--
		}
		for i := 0; i < n; i++ {
			offsets[i] = pos
			g := gap
			if extra > 0 {
				g++
				extra--
			}
			pos += allocs[i] + l.spacing + g
		}
	}

	return offsets
}

// shrinkToFit proportionally reduces allocations so they sum to at most target.
func (l *Layout) shrinkToFit(allocs []int, target int) {
	if target <= 0 {
		for i := range allocs {
			allocs[i] = 0
		}
		return
	}

	total := 0
	for _, a := range allocs {
		total += a
	}
	if total <= target {
		return
	}

	// Proportional shrink.
	for i := range allocs {
		allocs[i] = allocs[i] * target / total
	}

	// Fix rounding: give remainder to last non-zero.
	newTotal := 0
	for _, a := range allocs {
		newTotal += a
	}
	diff := target - newTotal
	for i := len(allocs) - 1; i >= 0 && diff > 0; i-- {
		allocs[i] += diff
		diff = 0
	}
}

// axisSize returns the size of rect along the layout direction.
func (l *Layout) axisSize(r Rect) int {
	if l.direction == Horizontal {
		return r.Width
	}
	return r.Height
}

// makeEmptyRects returns n zero-size rects positioned at the inner area's origin.
func makeEmptyRects(n int, inner Rect) []Rect {
	rects := make([]Rect, n)
	for i := range rects {
		rects[i] = Rect{X: inner.X, Y: inner.Y, Width: 0, Height: 0}
	}
	return rects
}

func clampNonNeg(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func clampRange(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
