package banner

import (
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// bnPlacement describes where a widget is placed on the character grid.
type bnPlacement struct {
	Widget WidgetData
	X      int // column offset (0-based)
	Y      int // row offset (0-based)
	W      int // allocated width (including border)
	H      int // allocated height (including border)
}

// bnColumnLayout describes the column configuration for a layout mode.
type bnColumnLayout struct {
	// Columns is the number of columns.
	Columns int
	// ColWidths returns the width of each column given total width.
	ColWidths func(totalWidth int) []int
	// WaifuCol is the column index for waifu widgets (or -1 if no
	// dedicated waifu column).
	WaifuCol int
}

// bnLayoutForPreset returns the column layout configuration for a given preset.
func bnLayoutForPreset(preset Preset) bnColumnLayout {
	switch preset.Name {
	case "compact":
		return bnColumnLayout{
			Columns: 1,
			ColWidths: func(w int) []int {
				return []int{w}
			},
			WaifuCol: -1,
		}
	case "standard":
		return bnColumnLayout{
			Columns: 2,
			ColWidths: func(w int) []int {
				left := w * 40 / 100 // 40% for waifu
				right := w - left
				return []int{left, right}
			},
			WaifuCol: 0,
		}
	case "wide":
		return bnColumnLayout{
			Columns: 3,
			ColWidths: func(w int) []int {
				left := w * 30 / 100   // 30% waifu
				mid := w * 35 / 100    // 35% data
				right := w - left - mid // remainder
				return []int{left, mid, right}
			},
			WaifuCol: 0,
		}
	case "ultrawide":
		return bnColumnLayout{
			Columns: 3,
			ColWidths: func(w int) []int {
				left := w * 35 / 100   // 35% wider waifu
				mid := w * 33 / 100    // 33% data
				right := w - left - mid // remainder
				return []int{left, mid, right}
			},
			WaifuCol: 0,
		}
	default:
		// Fallback to compact.
		return bnColumnLayout{
			Columns: 1,
			ColWidths: func(w int) []int {
				return []int{w}
			},
			WaifuCol: -1,
		}
	}
}

// bnArrangeWidgets determines placement of each widget using a greedy
// column-packing algorithm. It fills columns left-to-right, stacking
// widgets vertically within each column. Waifu widgets (ID starting with
// "waifu") go into the dedicated waifu column when available.
func bnArrangeWidgets(widgets []WidgetData, width, height int) []bnPlacement {
	if len(widgets) == 0 || width <= 0 || height <= 0 {
		return nil
	}

	// Determine preset from dimensions to get layout config.
	preset := SelectPreset(width, height)
	layout := bnLayoutForPreset(preset)
	colWidths := layout.ColWidths(width)

	// Track vertical cursor per column.
	colY := make([]int, layout.Columns)

	// Compute column X offsets.
	colX := make([]int, layout.Columns)
	offset := 0
	for i, w := range colWidths {
		colX[i] = offset
		offset += w
	}

	var placements []bnPlacement

	// Separate waifu widgets from data widgets.
	var waifuWidgets, dataWidgets []WidgetData
	for _, w := range widgets {
		if bnIsWaifuWidget(w) && layout.WaifuCol >= 0 {
			waifuWidgets = append(waifuWidgets, w)
		} else {
			dataWidgets = append(dataWidgets, w)
		}
	}

	// Place waifu widgets in the waifu column.
	if layout.WaifuCol >= 0 {
		for _, w := range waifuWidgets {
			col := layout.WaifuCol
			ww := colWidths[col]
			wh := bnWidgetHeight(w, ww, height-colY[col])
			if colY[col]+wh > height {
				wh = height - colY[col]
			}
			if wh <= 0 {
				continue
			}
			placements = append(placements, bnPlacement{
				Widget: w,
				X:      colX[col],
				Y:      colY[col],
				W:      ww,
				H:      wh,
			})
			colY[col] += wh
		}
	}

	// Place data widgets in remaining columns using greedy packing.
	for _, w := range dataWidgets {
		col := bnPickColumn(colY, layout, colWidths)
		ww := colWidths[col]
		wh := bnWidgetHeight(w, ww, height-colY[col])
		if colY[col]+wh > height {
			wh = height - colY[col]
		}
		if wh <= 0 {
			continue
		}
		placements = append(placements, bnPlacement{
			Widget: w,
			X:      colX[col],
			Y:      colY[col],
			W:      ww,
			H:      wh,
		})
		colY[col] += wh
	}

	return placements
}

// bnIsWaifuWidget returns true if the widget should be placed in the waifu column.
func bnIsWaifuWidget(w WidgetData) bool {
	return len(w.ID) >= 5 && w.ID[:5] == "waifu"
}

// bnWidgetHeight determines the rendered height for a widget. It uses the
// widget's MinH if set, otherwise calculates from content lines, capped by
// available space. The height includes 2 rows for the border.
func bnWidgetHeight(w WidgetData, colWidth, availHeight int) int {
	if availHeight <= 0 {
		return 0
	}

	minH := w.MinH
	if minH <= 0 {
		// Count content lines + 2 for border.
		lines := 1
		for i := range w.Content {
			if w.Content[i] == '\n' {
				lines++
			}
		}
		minH = lines + 2 // +2 for top and bottom border
	}

	if minH > availHeight {
		minH = availHeight
	}
	return minH
}

// bnPickColumn selects the column with the least vertical usage, excluding
// the waifu column.
func bnPickColumn(colY []int, layout bnColumnLayout, colWidths []int) int {
	best := -1
	bestY := int(^uint(0) >> 1) // max int
	for i := range colY {
		if i == layout.WaifuCol {
			continue
		}
		if colY[i] < bestY {
			bestY = colY[i]
			best = i
		}
	}
	if best < 0 {
		// All columns are waifu columns (shouldn't happen), use first.
		return 0
	}
	return best
}

// bnRenderWidgetBox wraps widget content in a bordered box at the given
// dimensions.
func bnRenderWidgetBox(w WidgetData, boxW, boxH int) string {
	style := components.DefaultBoxStyle()
	style.Title = w.Title
	return components.RenderBox(w.Content, boxW, boxH, style)
}
