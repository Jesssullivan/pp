package preset

import "sort"

// statusBarHeight is the number of rows reserved for the status bar.
const statusBarHeight = 1

// minWidgetWidth is the minimum usable width for any widget cell.
const minWidgetWidth = 10

// minWidgetHeight is the minimum usable height for any widget cell.
const minWidgetHeight = 3

// Resolve takes a preset and terminal dimensions, returns concrete cell
// positions. It distributes width evenly among columns (respecting ColSpan),
// distributes height among rows within each column based on RowSpan ratios,
// reserves 1 row for the status bar, and drops low-priority widgets when
// space is insufficient.
func Resolve(preset LayoutPreset, width, height int) []ResolvedCell {
	if width <= 0 || height <= 0 || len(preset.Widgets) == 0 {
		return nil
	}

	columns := preset.Columns
	if columns <= 0 {
		columns = 1
	}

	// Reserve status bar.
	availHeight := height - statusBarHeight
	if availHeight < minWidgetHeight {
		return nil
	}

	// Normalize all slots.
	slots := make([]WidgetSlot, len(preset.Widgets))
	for i, s := range preset.Widgets {
		slots[i] = prNormalizeSlot(s)
	}

	// Determine how many widget cells can fit and drop low-priority ones
	// if needed.
	maxCells := prMaxFittingCells(columns, width, availHeight)
	if maxCells < len(slots) {
		slots = prFilterByPriority(slots, maxCells)
	}

	if len(slots) == 0 {
		return nil
	}

	// Compute column widths.
	colWidths := prDistributeWidth(columns, width)

	// Group slots by column, then compute row allocations per column.
	colSlots := prGroupByColumn(slots, columns)

	cells := make([]ResolvedCell, 0, len(slots))

	for col := 0; col < columns; col++ {
		slotsInCol := colSlots[col]
		if len(slotsInCol) == 0 {
			continue
		}

		// Compute X offset for this column.
		xOffset := 0
		for c := 0; c < col; c++ {
			xOffset += colWidths[c]
		}

		// Compute the width of this cell (accounting for ColSpan).
		// For multi-column spans, sum the widths of spanned columns.
		rowAllocs := prDistributeHeight(slotsInCol, availHeight)

		for i, s := range slotsInCol {
			cellW := 0
			for c := s.Column; c < s.Column+s.ColSpan && c < columns; c++ {
				cellW += colWidths[c]
			}

			cells = append(cells, ResolvedCell{
				WidgetID: s.WidgetID,
				X:        xOffset,
				Y:        rowAllocs[i].y,
				W:        cellW,
				H:        rowAllocs[i].h,
			})
		}
	}

	return cells
}

// prRowAlloc holds a computed row Y position and height.
type prRowAlloc struct {
	y int
	h int
}

// prDistributeWidth returns column widths distributed evenly across totalWidth.
// Remainder pixels are distributed one per column from the left.
func prDistributeWidth(columns, totalWidth int) []int {
	if columns <= 0 {
		return nil
	}
	widths := make([]int, columns)
	base := totalWidth / columns
	remainder := totalWidth % columns

	for i := 0; i < columns; i++ {
		widths[i] = base
		if i < remainder {
			widths[i]++
		}
	}
	return widths
}

// prDistributeHeight distributes columnHeight among the given slots based on
// their RowSpan values as proportional weights. Slots are processed in row
// order. Returns a prRowAlloc for each slot with computed y and height.
func prDistributeHeight(slots []WidgetSlot, columnHeight int) []prRowAlloc {
	if len(slots) == 0 {
		return nil
	}

	// Sort slots by row to process top-to-bottom.
	sorted := make([]WidgetSlot, len(slots))
	copy(sorted, slots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Row < sorted[j].Row
	})

	// Sum all RowSpan values to determine total weight.
	totalWeight := 0
	for _, s := range sorted {
		totalWeight += s.RowSpan
	}

	if totalWeight <= 0 {
		totalWeight = len(sorted)
	}

	allocs := make([]prRowAlloc, len(sorted))
	y := 0
	allocated := 0

	for i, s := range sorted {
		weight := s.RowSpan
		if weight <= 0 {
			weight = 1
		}

		var h int
		if i == len(sorted)-1 {
			// Last slot gets the remainder to avoid rounding drift.
			h = columnHeight - allocated
		} else {
			h = columnHeight * weight / totalWeight
		}

		if h < 0 {
			h = 0
		}

		allocs[i] = prRowAlloc{y: y, h: h}
		y += h
		allocated += h
	}

	return allocs
}

// prGroupByColumn groups widget slots by their column index.
// Returns a slice of length columns, each containing the slots for that column.
func prGroupByColumn(slots []WidgetSlot, columns int) [][]WidgetSlot {
	groups := make([][]WidgetSlot, columns)
	for _, s := range slots {
		col := s.Column
		if col < 0 {
			col = 0
		}
		if col >= columns {
			col = columns - 1
		}
		groups[col] = append(groups[col], s)
	}
	return groups
}

// prMaxFittingCells estimates the maximum number of widget cells that can
// reasonably fit given the terminal dimensions.
func prMaxFittingCells(columns, width, height int) int {
	if columns <= 0 || width <= 0 || height <= 0 {
		return 0
	}
	colWidth := width / columns
	maxPerCol := height / minWidgetHeight
	if colWidth < minWidgetWidth {
		// Not enough width for even one column of widgets.
		return maxPerCol
	}
	return columns * maxPerCol
}
