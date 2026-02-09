package components

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Column alignment
// ---------------------------------------------------------------------------

// ColumnAlign controls horizontal text alignment within a table cell.
type ColumnAlign int

const (
	ColAlignLeft ColumnAlign = iota
	ColAlignCenter
	ColAlignRight
)

// ---------------------------------------------------------------------------
// Column sizing
// ---------------------------------------------------------------------------

// SizingKind discriminates the three column sizing strategies.
type SizingKind int

const (
	sizingFixed   SizingKind = iota
	sizingPercent            // percentage of total width
	sizingFill               // takes remaining space
)

// ColumnSizing describes how a column's width is computed.
type ColumnSizing struct {
	Kind  SizingKind
	Value int // width for Fixed, percentage 1-100 for Percent, unused for Fill
}

// SizingFixed returns a ColumnSizing that allocates exactly width characters.
func SizingFixed(width int) ColumnSizing {
	if width < 0 {
		width = 0
	}
	return ColumnSizing{Kind: sizingFixed, Value: width}
}

// SizingPercent returns a ColumnSizing that allocates pct% of available width.
func SizingPercent(pct int) ColumnSizing {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return ColumnSizing{Kind: sizingPercent, Value: pct}
}

// SizingFill returns a ColumnSizing that shares remaining space equally with
// other Fill columns.
func SizingFill() ColumnSizing {
	return ColumnSizing{Kind: sizingFill}
}

// ---------------------------------------------------------------------------
// Column and Row
// ---------------------------------------------------------------------------

// Column defines a single column in a DataTable.
type Column struct {
	Title    string
	Sizing   ColumnSizing
	Align    ColumnAlign
	MinWidth int
}

// Row represents a single data row in a DataTable.
type Row struct {
	Cells    []string
	Selected bool
	ID       string
}

// ---------------------------------------------------------------------------
// Style configuration
// ---------------------------------------------------------------------------

// HeaderStyleConfig controls the visual appearance of the header row.
type HeaderStyleConfig struct {
	Bold    bool
	FgColor string // hex "#RRGGBB"
	BgColor string // hex "#RRGGBB"
}

// RowStyleConfig controls the visual appearance of data rows.
type RowStyleConfig struct {
	EvenBgColor    string // hex "#RRGGBB" – even row (0-indexed) background
	OddBgColor     string // hex "#RRGGBB" – odd row background
	SelectedBgColor string // hex "#RRGGBB"
}

// DataTableConfig is the configuration used to construct a DataTable.
type DataTableConfig struct {
	Columns       []Column
	HeaderStyle   HeaderStyleConfig
	RowStyle      RowStyleConfig
	ShowHeader    bool
	ShowBorder    bool
	Selectable    bool
	BorderChar    string
	HeaderSepChar string
}

// ---------------------------------------------------------------------------
// DataTable
// ---------------------------------------------------------------------------

// DataTable is a scrollable, filterable, selectable table component.
type DataTable struct {
	mu           sync.Mutex
	columns      []Column
	rows         []Row
	headerStyle  HeaderStyleConfig
	rowStyle     RowStyleConfig
	showHeader   bool
	showBorder   bool
	selectable   bool
	borderChar   string
	headerSep    string
	scrollOffset int
	selectedIdx  int // index into filteredRows
	frozen       bool
	filterFn     func(Row) bool
	filteredRows []Row // cached filtered view
}

// NewDataTable creates a new DataTable from cfg. ShowHeader and ShowBorder
// default to true in Go's zero-value sense -- callers should explicitly set
// them to true when desired.
func NewDataTable(cfg DataTableConfig) *DataTable {
	border := cfg.BorderChar
	if border == "" {
		border = "│"
	}
	sep := cfg.HeaderSepChar
	if sep == "" {
		sep = "─"
	}

	dt := &DataTable{
		columns:     cfg.Columns,
		headerStyle: cfg.HeaderStyle,
		rowStyle:    cfg.RowStyle,
		showHeader:  cfg.ShowHeader,
		showBorder:  cfg.ShowBorder,
		selectable:  cfg.Selectable,
		borderChar:  border,
		headerSep:   sep,
		selectedIdx: -1,
	}
	dt.filteredRows = dt.applyFilter(dt.rows)
	return dt
}

// SetShowHeader sets whether the header row is displayed.
func (dt *DataTable) SetShowHeader(v bool) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.showHeader = v
}

// SetShowBorder sets whether column separators are displayed.
func (dt *DataTable) SetShowBorder(v bool) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.showBorder = v
}

// SetRows replaces all data. Resets scroll and selection.
func (dt *DataTable) SetRows(rows []Row) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.frozen {
		return
	}
	dt.rows = rows
	dt.filteredRows = dt.applyFilter(dt.rows)
	dt.scrollOffset = 0
	dt.selectedIdx = -1
}

// AppendRow adds a row at the end of the table.
func (dt *DataTable) AppendRow(row Row) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.frozen {
		return
	}
	dt.rows = append(dt.rows, row)
	dt.filteredRows = dt.applyFilter(dt.rows)
}

// ClearRows removes all rows.
func (dt *DataTable) ClearRows() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.frozen {
		return
	}
	dt.rows = nil
	dt.filteredRows = nil
	dt.scrollOffset = 0
	dt.selectedIdx = -1
}

// ScrollUp moves the viewport up by n rows.
func (dt *DataTable) ScrollUp(n int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.scrollOffset -= n
	if dt.scrollOffset < 0 {
		dt.scrollOffset = 0
	}
}

// ScrollDown moves the viewport down by n rows.
func (dt *DataTable) ScrollDown(n int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.scrollOffset += n
	max := len(dt.filteredRows)
	if dt.scrollOffset > max {
		dt.scrollOffset = max
	}
}

// ScrollToTop scrolls to the first row.
func (dt *DataTable) ScrollToTop() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.scrollOffset = 0
}

// ScrollToBottom scrolls so the last row is visible.
func (dt *DataTable) ScrollToBottom() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.scrollOffset = len(dt.filteredRows) // clamped during render
}

// SelectNext moves the selection cursor down.
func (dt *DataTable) SelectNext() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if !dt.selectable || len(dt.filteredRows) == 0 {
		return
	}
	dt.selectedIdx++
	if dt.selectedIdx >= len(dt.filteredRows) {
		dt.selectedIdx = len(dt.filteredRows) - 1
	}
}

// SelectPrev moves the selection cursor up.
func (dt *DataTable) SelectPrev() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if !dt.selectable || len(dt.filteredRows) == 0 {
		return
	}
	if dt.selectedIdx < 0 {
		dt.selectedIdx = 0
		return
	}
	dt.selectedIdx--
	if dt.selectedIdx < 0 {
		dt.selectedIdx = 0
	}
}

// SelectedRow returns the currently selected row, or nil if nothing is
// selected.
func (dt *DataTable) SelectedRow() *Row {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if !dt.selectable || dt.selectedIdx < 0 || dt.selectedIdx >= len(dt.filteredRows) {
		return nil
	}
	r := dt.filteredRows[dt.selectedIdx]
	return &r
}

// SetFilter installs a filter function. Pass nil to clear. Resets scroll and
// selection.
func (dt *DataTable) SetFilter(fn func(Row) bool) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.filterFn = fn
	dt.filteredRows = dt.applyFilter(dt.rows)
	dt.scrollOffset = 0
	dt.selectedIdx = -1
}

// Freeze prevents data mutations (SetRows, AppendRow, ClearRows) until
// Unfreeze is called. Useful during interactive scrolling.
func (dt *DataTable) Freeze() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.frozen = true
}

// Unfreeze re-enables data mutations.
func (dt *DataTable) Unfreeze() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.frozen = false
}

// IsFrozen returns whether the table is currently frozen.
func (dt *DataTable) IsFrozen() bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.frozen
}

// Render draws the table into a string of the given dimensions. Each line is
// exactly width visible characters (padded with spaces). The output has
// exactly height lines separated by newlines.
func (dt *DataTable) Render(width, height int) string {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if width <= 0 || height <= 0 {
		return ""
	}

	resetSeq := "\x1b[0m"

	// Resolve column widths.
	colWidths := dt.resolveWidths(width)

	// Determine how many header lines we need.
	headerLines := 0
	if dt.showHeader {
		headerLines = 2 // header row + separator
	}

	dataHeight := height - headerLines
	if dataHeight < 0 {
		dataHeight = 0
	}

	rows := dt.filteredRows

	// Handle empty table.
	if len(rows) == 0 && dataHeight > 0 {
		var lines []string
		if dt.showHeader {
			lines = append(lines, dt.renderHeader(colWidths, width))
			lines = append(lines, dt.renderSeparator(colWidths, width))
		}
		noData := "(no data)"
		if dtVisibleLen(noData) > width {
			noData = dtTruncateVisible(noData, width)
		}
		centered := dtPadVisible(noData, width, ColAlignCenter)
		lines = append(lines, centered)
		// Fill remaining height.
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return strings.Join(lines[:height], "\n")
	}

	// Clamp scroll offset.
	if dt.scrollOffset > len(rows) {
		dt.scrollOffset = len(rows)
	}
	// Ensure we can show at least some rows.
	if dataHeight > 0 {
		// Account for scroll indicators.
		topIndicator := dt.scrollOffset > 0
		bottomIndicator := (dt.scrollOffset + dataHeight) < len(rows)
		visibleDataLines := dataHeight
		if topIndicator {
			visibleDataLines--
		}
		if bottomIndicator {
			visibleDataLines--
		}
		// If indicators eat all space, don't show them.
		if visibleDataLines <= 0 {
			topIndicator = false
			bottomIndicator = false
			visibleDataLines = dataHeight
		}

		// Re-check with adjusted sizes.
		if topIndicator && !bottomIndicator {
			// Maybe bottom indicator is needed after adjustment.
			if dt.scrollOffset+visibleDataLines < len(rows) {
				bottomIndicator = true
				visibleDataLines--
				if visibleDataLines <= 0 {
					topIndicator = false
					bottomIndicator = false
					visibleDataLines = dataHeight
				}
			}
		}

		maxOffset := len(rows) - visibleDataLines
		if maxOffset < 0 {
			maxOffset = 0
		}
		if dt.scrollOffset > maxOffset {
			dt.scrollOffset = maxOffset
		}

		// Recalculate indicators after clamping.
		topIndicator = dt.scrollOffset > 0
		remaining := len(rows) - dt.scrollOffset
		visibleDataLines = dataHeight
		if topIndicator {
			visibleDataLines--
		}
		if remaining > visibleDataLines {
			bottomIndicator = true
			visibleDataLines--
		} else {
			bottomIndicator = false
		}
		if visibleDataLines <= 0 {
			topIndicator = false
			bottomIndicator = false
			visibleDataLines = dataHeight
		}

		var lines []string

		// Header.
		if dt.showHeader {
			lines = append(lines, dt.renderHeader(colWidths, width))
			lines = append(lines, dt.renderSeparator(colWidths, width))
		}

		// Top scroll indicator.
		if topIndicator {
			indicator := fmt.Sprintf("▲ %d more", dt.scrollOffset)
			if dtVisibleLen(indicator) > width {
				indicator = dtTruncateVisible(indicator, width)
			}
			lines = append(lines, dtPadVisible(indicator, width, ColAlignCenter))
		}

		// Data rows.
		end := dt.scrollOffset + visibleDataLines
		if end > len(rows) {
			end = len(rows)
		}
		for i := dt.scrollOffset; i < end; i++ {
			line := dt.renderRow(rows[i], i, colWidths, width)
			lines = append(lines, line+resetSeq)
		}

		// Bottom scroll indicator.
		if bottomIndicator {
			moreCount := len(rows) - end
			indicator := fmt.Sprintf("▼ %d more", moreCount)
			if dtVisibleLen(indicator) > width {
				indicator = dtTruncateVisible(indicator, width)
			}
			lines = append(lines, dtPadVisible(indicator, width, ColAlignCenter))
		}

		// Fill remaining height with blank lines.
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		if len(lines) > height {
			lines = lines[:height]
		}
		return strings.Join(lines, "\n")
	}

	// dataHeight == 0: header only.
	var lines []string
	if dt.showHeader && height >= 1 {
		lines = append(lines, dt.renderHeader(colWidths, width))
		if height >= 2 {
			lines = append(lines, dt.renderSeparator(colWidths, width))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines[:height], "\n")
}

// ---------------------------------------------------------------------------
// Internal rendering helpers
// ---------------------------------------------------------------------------

func (dt *DataTable) renderHeader(colWidths []int, totalWidth int) string {
	var sb strings.Builder
	fgSeq := dtColor(dt.headerStyle.FgColor)
	bgSeq := dtBgColor(dt.headerStyle.BgColor)
	boldSeq := ""
	resetSeq := "\x1b[0m"
	if dt.headerStyle.Bold {
		boldSeq = "\x1b[1m"
	}

	prefix := bgSeq + fgSeq + boldSeq

	usedWidth := 0
	for i, col := range dt.columns {
		if i >= len(colWidths) {
			break
		}
		w := colWidths[i]
		if w <= 0 {
			continue
		}
		if i > 0 && dt.showBorder && totalWidth >= 20 {
			sb.WriteString(prefix)
			sb.WriteString(dt.borderChar)
			usedWidth++
		}
		title := col.Title
		title = dtTruncateVisible(title, w)
		title = dtPadVisible(title, w, col.Align)
		sb.WriteString(prefix)
		sb.WriteString(title)
		usedWidth += w
	}
	sb.WriteString(resetSeq)

	// Pad line to totalWidth.
	if usedWidth < totalWidth {
		sb.WriteString(strings.Repeat(" ", totalWidth-usedWidth))
	}
	result := sb.String()
	return dtTrimTrailingVisibleSpaces(result, totalWidth)
}

func (dt *DataTable) renderSeparator(colWidths []int, totalWidth int) string {
	var sb strings.Builder
	usedWidth := 0
	for i, w := range colWidths {
		if w <= 0 {
			continue
		}
		if i > 0 && dt.showBorder && totalWidth >= 20 {
			sb.WriteString("┼")
			usedWidth++
		}
		sb.WriteString(strings.Repeat(dt.headerSep, w))
		usedWidth += w
	}
	if usedWidth < totalWidth {
		sb.WriteString(strings.Repeat(dt.headerSep, totalWidth-usedWidth))
	}
	line := sb.String()
	// Truncate to totalWidth visible chars.
	if dtVisibleLen(line) > totalWidth {
		line = dtTruncateVisible(line, totalWidth)
	}
	return line
}

func (dt *DataTable) renderRow(row Row, rowIndex int, colWidths []int, totalWidth int) string {
	var sb strings.Builder
	resetSeq := "\x1b[0m"

	// Determine background.
	bgSeq := ""
	if dt.selectable && dt.selectedIdx >= 0 && rowIndex == dt.selectedIdx {
		bgSeq = dtBgColor(dt.rowStyle.SelectedBgColor)
	} else if rowIndex%2 == 0 {
		bgSeq = dtBgColor(dt.rowStyle.EvenBgColor)
	} else {
		bgSeq = dtBgColor(dt.rowStyle.OddBgColor)
	}

	usedWidth := 0
	for i, col := range dt.columns {
		if i >= len(colWidths) {
			break
		}
		w := colWidths[i]
		if w <= 0 {
			continue
		}
		if i > 0 && dt.showBorder && totalWidth >= 20 {
			sb.WriteString(bgSeq)
			sb.WriteString(dt.borderChar)
			usedWidth++
		}
		cell := ""
		if i < len(row.Cells) {
			cell = row.Cells[i]
		}
		cell = dtTruncateVisible(cell, w)
		cell = dtPadVisible(cell, w, col.Align)
		sb.WriteString(bgSeq)
		sb.WriteString(cell)
		usedWidth += w
	}

	// Pad to totalWidth.
	if usedWidth < totalWidth {
		sb.WriteString(bgSeq)
		sb.WriteString(strings.Repeat(" ", totalWidth-usedWidth))
	}
	sb.WriteString(resetSeq)

	return sb.String()
}

// ---------------------------------------------------------------------------
// Column width resolution (3-pass algorithm)
// ---------------------------------------------------------------------------

func (dt *DataTable) resolveWidths(totalWidth int) []int {
	n := len(dt.columns)
	if n == 0 {
		return nil
	}

	widths := make([]int, n)

	// Calculate separator overhead.
	sepOverhead := 0
	if dt.showBorder && totalWidth >= 20 {
		sepOverhead = n - 1 // one separator between each pair
	}
	available := totalWidth - sepOverhead
	if available < 0 {
		available = 0
	}

	// Pass 1: Fixed columns.
	remaining := available
	for i, col := range dt.columns {
		if col.Sizing.Kind == sizingFixed {
			w := col.Sizing.Value
			if w > remaining {
				w = remaining
			}
			widths[i] = w
			remaining -= w
		}
	}

	// Pass 2: Percentage columns.
	for i, col := range dt.columns {
		if col.Sizing.Kind == sizingPercent {
			w := (available * col.Sizing.Value) / 100
			if w > remaining {
				w = remaining
			}
			widths[i] = w
			remaining -= w
		}
	}

	// Pass 3: Fill columns share remaining space equally.
	fillCount := 0
	for _, col := range dt.columns {
		if col.Sizing.Kind == sizingFill {
			fillCount++
		}
	}
	if fillCount > 0 && remaining > 0 {
		each := remaining / fillCount
		extra := remaining % fillCount
		filled := 0
		for i, col := range dt.columns {
			if col.Sizing.Kind == sizingFill {
				w := each
				if filled < extra {
					w++
				}
				widths[i] = w
				filled++
			}
		}
	}

	// Pass 4: Enforce MinWidth constraints.
	for i, col := range dt.columns {
		if col.MinWidth > 0 && widths[i] < col.MinWidth {
			deficit := col.MinWidth - widths[i]
			widths[i] = col.MinWidth
			// Steal from Fill columns (right to left).
			for j := n - 1; j >= 0 && deficit > 0; j-- {
				if j == i {
					continue
				}
				if dt.columns[j].Sizing.Kind == sizingFill {
					canSteal := widths[j] - dt.columns[j].MinWidth
					if canSteal <= 0 {
						continue
					}
					steal := deficit
					if steal > canSteal {
						steal = canSteal
					}
					widths[j] -= steal
					deficit -= steal
				}
			}
		}
	}

	// Pass 5: If total exceeds available width, truncate from rightmost Fill.
	totalUsed := 0
	for _, w := range widths {
		totalUsed += w
	}
	if totalUsed > available {
		excess := totalUsed - available
		for i := n - 1; i >= 0 && excess > 0; i-- {
			if dt.columns[i].Sizing.Kind == sizingFill {
				canCut := widths[i]
				if dt.columns[i].MinWidth > 0 {
					canCut = widths[i] - dt.columns[i].MinWidth
				}
				if canCut <= 0 {
					continue
				}
				cut := excess
				if cut > canCut {
					cut = canCut
				}
				widths[i] -= cut
				excess -= cut
			}
		}
	}

	return widths
}

// ---------------------------------------------------------------------------
// Filter helper
// ---------------------------------------------------------------------------

func (dt *DataTable) applyFilter(rows []Row) []Row {
	if dt.filterFn == nil {
		// Return a copy to avoid aliasing.
		if len(rows) == 0 {
			return nil
		}
		out := make([]Row, len(rows))
		copy(out, rows)
		return out
	}
	var out []Row
	for _, r := range rows {
		if dt.filterFn(r) {
			out = append(out, r)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Private ANSI / string helpers (self-contained, no sibling imports)
// ---------------------------------------------------------------------------

// dtVisibleLen returns the number of visible characters in s, skipping ANSI
// escape sequences. Each rune counts as 1 (no wide-char handling in the
// self-contained version).
func dtVisibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

// dtTruncateVisible truncates s so its visible width is at most max. If
// truncation occurs, "…" is appended (consuming 1 visible char). ANSI
// sequences before the cut point are preserved.
func dtTruncateVisible(s string, max int) string {
	if max <= 0 {
		return ""
	}
	vis := dtVisibleLen(s)
	if vis <= max {
		return s
	}
	// We need to keep (max-1) visible chars and append "…".
	cutAt := max - 1
	if cutAt < 0 {
		cutAt = 0
	}

	var sb strings.Builder
	count := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			sb.WriteRune(r)
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			sb.WriteRune(r)
			continue
		}
		if count >= cutAt {
			break
		}
		sb.WriteRune(r)
		count++
	}
	sb.WriteString("…")
	return sb.String()
}

// dtPadVisible pads s with spaces to the given width according to align.
// If s is already wider than width, it is returned as-is.
func dtPadVisible(s string, width int, align ColumnAlign) string {
	vis := dtVisibleLen(s)
	if vis >= width {
		return s
	}
	pad := width - vis
	switch align {
	case ColAlignRight:
		return strings.Repeat(" ", pad) + s
	case ColAlignCenter:
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // ColAlignLeft
		return s + strings.Repeat(" ", pad)
	}
}

// dtColor returns an ANSI true-color foreground sequence from a "#RRGGBB"
// hex string. Returns "" for empty or invalid input.
func dtColor(hex string) string {
	r, g, b, ok := dtParseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// dtBgColor returns an ANSI true-color background sequence from a "#RRGGBB"
// hex string. Returns "" for empty or invalid input.
func dtBgColor(hex string) string {
	r, g, b, ok := dtParseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// dtParseHex parses "#RRGGBB" or "RRGGBB" into (r, g, b, ok).
func dtParseHex(hex string) (r, g, b uint8, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(rv), uint8(gv), uint8(bv), true
}

// dtTrimTrailingVisibleSpaces trims trailing whitespace but preserves ANSI
// reset sequences. The result is padded to targetWidth visible chars.
// In practice we just return the string as-is since the caller already
// manages width. This is a no-op placeholder for potential future use.
func dtTrimTrailingVisibleSpaces(s string, _ int) string {
	return s
}
