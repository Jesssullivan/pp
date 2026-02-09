package components

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// lineCount returns the number of lines in rendered output.
func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// lines splits rendered output into individual lines.
func lines(s string) []string {
	return strings.Split(s, "\n")
}

// containsVisible checks that the rendered output contains sub somewhere
// in visible text (ANSI stripped).
func containsVisible(rendered, sub string) bool {
	stripped := stripANSI(rendered)
	return strings.Contains(stripped, sub)
}

// stripANSI removes all ANSI CSI sequences from s.
func stripANSI(s string) string {
	var sb strings.Builder
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
		sb.WriteRune(r)
	}
	return sb.String()
}

// defaultCfg returns a simple 3-column config for testing.
func defaultCfg() DataTableConfig {
	return DataTableConfig{
		Columns: []Column{
			{Title: "Name", Sizing: SizingFill(), Align: ColAlignLeft},
			{Title: "Age", Sizing: SizingFixed(5), Align: ColAlignRight},
			{Title: "City", Sizing: SizingFill(), Align: ColAlignLeft},
		},
		ShowHeader: true,
		ShowBorder: true,
	}
}

// sampleRows returns a small set of test rows.
func sampleRows() []Row {
	return []Row{
		{ID: "1", Cells: []string{"Alice", "30", "New York"}},
		{ID: "2", Cells: []string{"Bob", "25", "London"}},
		{ID: "3", Cells: []string{"Charlie", "35", "Tokyo"}},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewDataTable(t *testing.T) {
	dt := NewDataTable(defaultCfg())
	if dt == nil {
		t.Fatal("NewDataTable returned nil")
	}
	if len(dt.columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(dt.columns))
	}
	if dt.borderChar != "│" {
		t.Errorf("expected default border char │, got %q", dt.borderChar)
	}
	if dt.headerSep != "─" {
		t.Errorf("expected default header sep ─, got %q", dt.headerSep)
	}
}

func TestColumnWidthFixed(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingFixed(10)},
			{Title: "B", Sizing: SizingFixed(20)},
		},
		ShowBorder: true,
		ShowHeader: true,
	}
	dt := NewDataTable(cfg)
	widths := dt.resolveWidths(40)
	if widths[0] != 10 {
		t.Errorf("col 0: expected 10, got %d", widths[0])
	}
	if widths[1] != 20 {
		t.Errorf("col 1: expected 20, got %d", widths[1])
	}
}

func TestColumnWidthPercent(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingPercent(50)},
			{Title: "B", Sizing: SizingPercent(50)},
		},
		ShowBorder: true,
		ShowHeader: true,
	}
	dt := NewDataTable(cfg)
	// totalWidth=41 -> available = 41 - 1 (one separator) = 40
	widths := dt.resolveWidths(41)
	if widths[0] != 20 {
		t.Errorf("col 0: expected 20, got %d", widths[0])
	}
	if widths[1] != 20 {
		t.Errorf("col 1: expected 20, got %d", widths[1])
	}
}

func TestColumnWidthFill(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingFill()},
			{Title: "B", Sizing: SizingFill()},
		},
		ShowBorder: true,
		ShowHeader: true,
	}
	dt := NewDataTable(cfg)
	// totalWidth=41 -> available = 40 -> each fill = 20
	widths := dt.resolveWidths(41)
	if widths[0] != 20 {
		t.Errorf("col 0: expected 20, got %d", widths[0])
	}
	if widths[1] != 20 {
		t.Errorf("col 1: expected 20, got %d", widths[1])
	}
}

func TestColumnWidthMixed(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Fixed", Sizing: SizingFixed(10)},
			{Title: "Pct", Sizing: SizingPercent(25)},
			{Title: "Fill", Sizing: SizingFill()},
		},
		ShowBorder: true,
		ShowHeader: true,
	}
	dt := NewDataTable(cfg)
	// totalWidth=50 -> available = 50 - 2 (two seps) = 48
	// Fixed: 10, remaining = 38
	// Pct(25% of 48) = 12, remaining = 26
	// Fill: 26
	widths := dt.resolveWidths(50)
	if widths[0] != 10 {
		t.Errorf("fixed: expected 10, got %d", widths[0])
	}
	if widths[1] != 12 {
		t.Errorf("pct: expected 12, got %d", widths[1])
	}
	if widths[2] != 26 {
		t.Errorf("fill: expected 26, got %d", widths[2])
	}
}

func TestMinWidthEnforcement(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Narrow", Sizing: SizingFixed(3), MinWidth: 8},
			{Title: "Fill", Sizing: SizingFill()},
		},
		ShowBorder: true,
		ShowHeader: true,
	}
	dt := NewDataTable(cfg)
	// totalWidth=40 -> available = 39 (one sep)
	// Fixed: 3 -> enforced to 8 (deficit 5, steal from Fill)
	// Fill: remaining = 39 - 3 = 36, then steal 5 -> 31
	widths := dt.resolveWidths(40)
	if widths[0] != 8 {
		t.Errorf("narrow: expected 8, got %d", widths[0])
	}
	if widths[1] != 31 {
		t.Errorf("fill: expected 31, got %d", widths[1])
	}
}

func TestHeaderRendering(t *testing.T) {
	cfg := defaultCfg()
	cfg.HeaderStyle = HeaderStyleConfig{
		Bold:    true,
		FgColor: "#ffffff",
		BgColor: "#000000",
	}
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(40, 10)
	if !containsVisible(out, "Name") {
		t.Error("header should contain 'Name'")
	}
	if !containsVisible(out, "Age") {
		t.Error("header should contain 'Age'")
	}
	if !containsVisible(out, "City") {
		t.Error("header should contain 'City'")
	}
	// Check bold sequence present.
	if !strings.Contains(out, "\x1b[1m") {
		t.Error("header should contain bold ANSI sequence")
	}
}

func TestHeaderWithHeaderStyleColors(t *testing.T) {
	cfg := defaultCfg()
	cfg.HeaderStyle = HeaderStyleConfig{
		FgColor: "#ff0000",
		BgColor: "#00ff00",
	}
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(40, 10)
	// Check fg color sequence.
	if !strings.Contains(out, "\x1b[38;2;255;0;0m") {
		t.Error("header should contain red foreground sequence")
	}
	// Check bg color sequence.
	if !strings.Contains(out, "\x1b[48;2;0;255;0m") {
		t.Error("header should contain green background sequence")
	}
}

func TestDataRowRenderingZebra(t *testing.T) {
	cfg := defaultCfg()
	cfg.RowStyle = RowStyleConfig{
		EvenBgColor: "#111111",
		OddBgColor:  "#222222",
	}
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(40, 10)
	// Even row bg.
	evenBg := "\x1b[48;2;17;17;17m" // #111111
	oddBg := "\x1b[48;2;34;34;34m"  // #222222
	if !strings.Contains(out, evenBg) {
		t.Error("should contain even row background color")
	}
	if !strings.Contains(out, oddBg) {
		t.Error("should contain odd row background color")
	}
}

func TestScrollDown(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	rows := make([]Row, 20)
	for i := range rows {
		rows[i] = Row{ID: fmt.Sprintf("%d", i), Cells: []string{fmt.Sprintf("Row%d", i), "0", "X"}}
	}
	dt.SetRows(rows)

	// Render in a small viewport: header(2) + data(3) = 5 lines.
	dt.ScrollDown(5)
	out := dt.Render(40, 5)
	// Should show scroll indicators.
	if !containsVisible(out, "▲") {
		t.Error("should show top scroll indicator after scrolling down")
	}
}

func TestScrollUp(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	rows := make([]Row, 20)
	for i := range rows {
		rows[i] = Row{ID: fmt.Sprintf("%d", i), Cells: []string{fmt.Sprintf("Row%d", i), "0", "X"}}
	}
	dt.SetRows(rows)

	dt.ScrollDown(10)
	dt.ScrollUp(5)
	out := dt.Render(40, 5)
	// Should still show top indicator since offset > 0.
	if !containsVisible(out, "▲") {
		t.Error("should show top scroll indicator")
	}
}

func TestScrollToTopAndBottom(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	rows := make([]Row, 20)
	for i := range rows {
		rows[i] = Row{ID: fmt.Sprintf("%d", i), Cells: []string{fmt.Sprintf("Row%d", i), "0", "X"}}
	}
	dt.SetRows(rows)

	dt.ScrollToBottom()
	out := dt.Render(40, 7)
	if !containsVisible(out, "▲") {
		t.Error("scroll to bottom should show top indicator")
	}

	dt.ScrollToTop()
	out = dt.Render(40, 7)
	// At top, no top indicator.
	stripped := stripANSI(out)
	if strings.Contains(stripped, "▲") {
		t.Error("scroll to top should not show top indicator")
	}
}

func TestScrollIndicatorCounts(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Name", Sizing: SizingFill()},
		},
		ShowHeader: true,
		ShowBorder: true,
	}
	dt := NewDataTable(cfg)
	rows := make([]Row, 10)
	for i := range rows {
		rows[i] = Row{ID: fmt.Sprintf("%d", i), Cells: []string{fmt.Sprintf("Item%d", i)}}
	}
	dt.SetRows(rows)

	// Height 5 = header(2) + data(3). With 10 rows, bottom indicator should show.
	out := dt.Render(30, 5)
	if !containsVisible(out, "▼") {
		t.Error("should show bottom scroll indicator")
	}
	// Check the count: data area = 3, with bottom indicator = 2 visible rows.
	// Remaining = 10 - 2 = 8 more.
	if !containsVisible(out, "8 more") {
		t.Errorf("should show '8 more' at bottom, got:\n%s", stripANSI(out))
	}
}

func TestSelectionNextPrev(t *testing.T) {
	cfg := defaultCfg()
	cfg.Selectable = true
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())

	dt.SelectNext()
	r := dt.SelectedRow()
	if r == nil {
		t.Fatal("selected row should not be nil after SelectNext")
	}
	if r.ID != "1" {
		t.Errorf("expected first row selected (id=1), got %s", r.ID)
	}

	dt.SelectNext()
	r = dt.SelectedRow()
	if r == nil || r.ID != "2" {
		t.Errorf("expected second row selected (id=2), got %v", r)
	}

	dt.SelectPrev()
	r = dt.SelectedRow()
	if r == nil || r.ID != "1" {
		t.Errorf("expected first row selected again (id=1), got %v", r)
	}

	// Clamp at top.
	dt.SelectPrev()
	dt.SelectPrev()
	dt.SelectPrev()
	r = dt.SelectedRow()
	if r == nil || r.ID != "1" {
		t.Error("selection should clamp at first row")
	}
}

func TestSelectionClampAtBottom(t *testing.T) {
	cfg := defaultCfg()
	cfg.Selectable = true
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())

	for i := 0; i < 10; i++ {
		dt.SelectNext()
	}
	r := dt.SelectedRow()
	if r == nil || r.ID != "3" {
		t.Error("selection should clamp at last row")
	}
}

func TestSelectionRendering(t *testing.T) {
	cfg := defaultCfg()
	cfg.Selectable = true
	cfg.RowStyle = RowStyleConfig{
		SelectedBgColor: "#ff0000",
	}
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())

	dt.SelectNext()
	out := dt.Render(40, 10)
	selectedBg := "\x1b[48;2;255;0;0m"
	if !strings.Contains(out, selectedBg) {
		t.Error("selected row should have red background")
	}
}

func TestFilter(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())

	dt.SetFilter(func(r Row) bool {
		return len(r.Cells) > 0 && r.Cells[0] == "Alice"
	})

	out := dt.Render(40, 10)
	if !containsVisible(out, "Alice") {
		t.Error("filtered output should contain Alice")
	}
	if containsVisible(out, "Bob") {
		t.Error("filtered output should not contain Bob")
	}
	if containsVisible(out, "Charlie") {
		t.Error("filtered output should not contain Charlie")
	}

	// Clear filter.
	dt.SetFilter(nil)
	out = dt.Render(40, 10)
	if !containsVisible(out, "Bob") {
		t.Error("after clearing filter, Bob should be visible")
	}
}

func TestFreezeUnfreeze(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())

	dt.Freeze()
	if !dt.IsFrozen() {
		t.Error("table should be frozen")
	}

	dt.SetRows([]Row{{ID: "x", Cells: []string{"NewRow", "0", "Nowhere"}}})
	out := dt.Render(40, 10)
	// Should still show old data.
	if containsVisible(out, "NewRow") {
		t.Error("frozen table should not accept SetRows")
	}
	if !containsVisible(out, "Alice") {
		t.Error("frozen table should still show Alice")
	}

	dt.Unfreeze()
	if dt.IsFrozen() {
		t.Error("table should be unfrozen")
	}
	dt.SetRows([]Row{{ID: "x", Cells: []string{"NewRow", "0", "Nowhere"}}})
	out = dt.Render(40, 10)
	if !containsVisible(out, "NewRow") {
		t.Error("unfrozen table should accept new data")
	}
}

func TestFreezeAppend(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	dt.Freeze()
	dt.AppendRow(Row{ID: "new", Cells: []string{"New", "99", "Mars"}})
	out := dt.Render(40, 10)
	if containsVisible(out, "Mars") {
		t.Error("frozen table should not accept AppendRow")
	}
}

func TestFreezeClear(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	dt.Freeze()
	dt.ClearRows()
	out := dt.Render(40, 10)
	if containsVisible(out, "(no data)") {
		t.Error("frozen table should not accept ClearRows")
	}
	if !containsVisible(out, "Alice") {
		t.Error("frozen table should still show data after ClearRows attempt")
	}
}

func TestTruncationWithEllipsis(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Name", Sizing: SizingFixed(6), Align: ColAlignLeft},
		},
		ShowHeader: true,
		ShowBorder: false,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{
		{ID: "1", Cells: []string{"VeryLongName"}},
	})
	out := dt.Render(6, 4)
	if !containsVisible(out, "…") {
		t.Error("long cell content should be truncated with …")
	}
}

func TestEmptyTableNoData(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	out := dt.Render(40, 5)
	if !containsVisible(out, "(no data)") {
		t.Errorf("empty table should show '(no data)', got:\n%s", stripANSI(out))
	}
}

func TestRenderSize20x5(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(20, 5)
	if lineCount(out) != 5 {
		t.Errorf("expected 5 lines, got %d", lineCount(out))
	}
}

func TestRenderSize40x10(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(40, 10)
	if lineCount(out) != 10 {
		t.Errorf("expected 10 lines, got %d", lineCount(out))
	}
	if !containsVisible(out, "Alice") {
		t.Error("should contain Alice at 40x10")
	}
}

func TestRenderSize80x24(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(80, 24)
	if lineCount(out) != 24 {
		t.Errorf("expected 24 lines, got %d", lineCount(out))
	}
}

func TestRenderSize120x30(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(120, 30)
	if lineCount(out) != 30 {
		t.Errorf("expected 30 lines, got %d", lineCount(out))
	}
}

func TestColumnAlignLeft(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Name", Sizing: SizingFixed(10), Align: ColAlignLeft},
		},
		ShowHeader: true,
		ShowBorder: false,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: []string{"Hi"}}})
	out := dt.Render(10, 4)
	ls := lines(out)
	// Data row (line index 2) should start with "Hi" followed by spaces.
	dataLine := stripANSI(ls[2])
	if !strings.HasPrefix(dataLine, "Hi") {
		t.Errorf("left-aligned cell should start with content, got %q", dataLine)
	}
}

func TestColumnAlignRight(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Num", Sizing: SizingFixed(10), Align: ColAlignRight},
		},
		ShowHeader: true,
		ShowBorder: false,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: []string{"42"}}})
	out := dt.Render(10, 4)
	ls := lines(out)
	dataLine := stripANSI(ls[2])
	if !strings.HasSuffix(strings.TrimRight(dataLine, " "), "42") {
		t.Errorf("right-aligned cell should end with content, got %q", dataLine)
	}
}

func TestColumnAlignCenter(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Mid", Sizing: SizingFixed(10), Align: ColAlignCenter},
		},
		ShowHeader: true,
		ShowBorder: false,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: []string{"Hi"}}})
	out := dt.Render(10, 4)
	ls := lines(out)
	dataLine := stripANSI(ls[2])
	// "Hi" is 2 chars wide, column is 10 -> 4 spaces left, 4 spaces right.
	if !strings.HasPrefix(dataLine, "    Hi") {
		t.Errorf("center-aligned cell should be centered, got %q", dataLine)
	}
}

func TestSingleColumn(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Only", Sizing: SizingFill()},
		},
		ShowHeader: true,
		ShowBorder: true,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{
		{ID: "1", Cells: []string{"Hello"}},
	})
	out := dt.Render(20, 5)
	if !containsVisible(out, "Hello") {
		t.Error("single column table should render cell content")
	}
	if !containsVisible(out, "Only") {
		t.Error("single column table should render header")
	}
}

func TestManyColumns(t *testing.T) {
	cols := make([]Column, 12)
	cells := make([]string, 12)
	for i := range cols {
		cols[i] = Column{
			Title:  fmt.Sprintf("C%d", i),
			Sizing: SizingFill(),
		}
		cells[i] = fmt.Sprintf("v%d", i)
	}
	cfg := DataTableConfig{
		Columns:    cols,
		ShowHeader: true,
		ShowBorder: true,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: cells}})
	out := dt.Render(80, 5)
	if lineCount(out) != 5 {
		t.Errorf("expected 5 lines, got %d", lineCount(out))
	}
	// At least some column headers should be visible.
	if !containsVisible(out, "C0") {
		t.Error("first column header should be visible")
	}
}

func TestNoHeaderMode(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "Name", Sizing: SizingFill()},
		},
		ShowHeader: false,
		ShowBorder: true,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: []string{"Alice"}}})
	out := dt.Render(20, 5)
	if containsVisible(out, "Name") {
		t.Error("header should not be visible when ShowHeader is false")
	}
	if !containsVisible(out, "Alice") {
		t.Error("data should still be visible when ShowHeader is false")
	}
}

func TestHeaderSeparatorChar(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingFill()},
		},
		ShowHeader:    true,
		ShowBorder:    true,
		HeaderSepChar: "=",
	}
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(20, 5)
	ls := lines(out)
	// Second line should be the separator.
	if len(ls) >= 2 && !strings.Contains(ls[1], "=") {
		t.Errorf("separator should use custom char '=', got %q", ls[1])
	}
}

func TestBorderSeparator(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(40, 10)
	if !containsVisible(out, "│") {
		t.Error("should show column separator │")
	}
	if !containsVisible(out, "┼") {
		t.Error("should show separator crossing ┼")
	}
}

func TestNoBorderMode(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingFill()},
			{Title: "B", Sizing: SizingFill()},
		},
		ShowHeader: true,
		ShowBorder: false,
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: []string{"X", "Y"}}})
	out := dt.Render(40, 5)
	if containsVisible(out, "│") {
		t.Error("should not show border chars when ShowBorder is false")
	}
}

func TestHeightLessThan3HeaderOnly(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	out := dt.Render(40, 2)
	if lineCount(out) != 2 {
		t.Errorf("expected 2 lines, got %d", lineCount(out))
	}
	if !containsVisible(out, "Name") {
		t.Error("should show header even at height 2")
	}
}

func TestZeroWidthHeight(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	if out := dt.Render(0, 10); out != "" {
		t.Error("zero width should produce empty output")
	}
	if out := dt.Render(10, 0); out != "" {
		t.Error("zero height should produce empty output")
	}
}

func TestSelectedRowNilWhenNotSelectable(t *testing.T) {
	cfg := defaultCfg()
	cfg.Selectable = false
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	if dt.SelectedRow() != nil {
		t.Error("SelectedRow should be nil when not selectable")
	}
}

func TestSelectedRowNilWhenEmpty(t *testing.T) {
	cfg := defaultCfg()
	cfg.Selectable = true
	dt := NewDataTable(cfg)
	dt.SelectNext()
	if dt.SelectedRow() != nil {
		t.Error("SelectedRow should be nil when no rows exist")
	}
}

func TestAppendRow(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.AppendRow(Row{ID: "1", Cells: []string{"Alice", "30", "NYC"}})
	dt.AppendRow(Row{ID: "2", Cells: []string{"Bob", "25", "LA"}})
	out := dt.Render(40, 10)
	if !containsVisible(out, "Alice") {
		t.Error("should contain appended row Alice")
	}
	if !containsVisible(out, "Bob") {
		t.Error("should contain appended row Bob")
	}
}

func TestClearRows(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	dt.ClearRows()
	out := dt.Render(40, 5)
	if !containsVisible(out, "(no data)") {
		t.Error("cleared table should show (no data)")
	}
}

func TestGracefulDegradationNarrowWidth(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	// Width < 20: should hide column separators.
	out := dt.Render(15, 5)
	if containsVisible(out, "│") {
		t.Error("narrow width (<20) should hide column separators")
	}
	if lineCount(out) != 5 {
		t.Errorf("expected 5 lines even at narrow width, got %d", lineCount(out))
	}
}

// ---------------------------------------------------------------------------
// Private helper tests
// ---------------------------------------------------------------------------

func TestDtVisibleLen(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"\x1b[1mhello\x1b[0m", 5},
		{"\x1b[38;2;255;0;0mred\x1b[0m", 3},
		{"", 0},
		{"no ansi", 7},
	}
	for _, tt := range tests {
		got := dtVisibleLen(tt.input)
		if got != tt.want {
			t.Errorf("dtVisibleLen(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDtTruncateVisible(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  int // expected visible length of result
	}{
		{"hello", 3, 3},        // "he…"
		{"hello", 10, 5},       // no truncation
		{"hello", 1, 1},        // "…"
		{"hello", 0, 0},        // empty
		{"\x1b[1mhello\x1b[0m", 3, 3}, // with ANSI
	}
	for _, tt := range tests {
		got := dtTruncateVisible(tt.input, tt.max)
		gotLen := dtVisibleLen(got)
		if gotLen != tt.want {
			t.Errorf("dtTruncateVisible(%q, %d) visible len = %d, want %d (result: %q)",
				tt.input, tt.max, gotLen, tt.want, got)
		}
	}
}

func TestDtPadVisible(t *testing.T) {
	tests := []struct {
		input string
		width int
		align ColumnAlign
		want  string
	}{
		{"hi", 6, ColAlignLeft, "hi    "},
		{"hi", 6, ColAlignRight, "    hi"},
		{"hi", 6, ColAlignCenter, "  hi  "},
		{"hi", 2, ColAlignLeft, "hi"},
		{"hi", 1, ColAlignLeft, "hi"}, // wider than width, returned as-is
	}
	for _, tt := range tests {
		got := dtPadVisible(tt.input, tt.width, tt.align)
		if got != tt.want {
			t.Errorf("dtPadVisible(%q, %d, %v) = %q, want %q",
				tt.input, tt.width, tt.align, got, tt.want)
		}
	}
}

func TestDtParseHex(t *testing.T) {
	tests := []struct {
		input string
		r, g, b uint8
		ok      bool
	}{
		{"#ff0000", 255, 0, 0, true},
		{"00ff00", 0, 255, 0, true},
		{"#0000ff", 0, 0, 255, true},
		{"invalid", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"#fff", 0, 0, 0, false}, // short hex not supported
	}
	for _, tt := range tests {
		r, g, b, ok := dtParseHex(tt.input)
		if ok != tt.ok || r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("dtParseHex(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tt.input, r, g, b, ok, tt.r, tt.g, tt.b, tt.ok)
		}
	}
}

func TestDataTableRenderOutputLineCount(t *testing.T) {
	// Verify all output sizes produce exactly height lines.
	sizes := [][2]int{{20, 5}, {40, 10}, {80, 24}, {120, 30}, {15, 3}}
	cfg := defaultCfg()
	for _, sz := range sizes {
		dt := NewDataTable(cfg)
		dt.SetRows(sampleRows())
		out := dt.Render(sz[0], sz[1])
		if lc := lineCount(out); lc != sz[1] {
			t.Errorf("Render(%d, %d): expected %d lines, got %d",
				sz[0], sz[1], sz[1], lc)
		}
	}
}

func TestColumnWidthFillExtraDistribution(t *testing.T) {
	// 3 fill columns with 40 available (no borders, width < 20).
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingFill()},
			{Title: "B", Sizing: SizingFill()},
			{Title: "C", Sizing: SizingFill()},
		},
		ShowBorder: false,
		ShowHeader: true,
	}
	dt := NewDataTable(cfg)
	widths := dt.resolveWidths(10) // no border since width < 20 is handled in render, but resolveWidths uses showBorder directly
	// No border overhead since ShowBorder=false. available = 10.
	// 10 / 3 = 3 each, 1 extra -> first gets 4.
	total := 0
	for _, w := range widths {
		total += w
	}
	if total != 10 {
		t.Errorf("fill columns should sum to available width 10, got %d", total)
	}
}

func TestScrollUpBeyondZero(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	dt.ScrollUp(100)
	if dt.scrollOffset != 0 {
		t.Errorf("scrollOffset should clamp to 0, got %d", dt.scrollOffset)
	}
}

func TestSizingPercentClamped(t *testing.T) {
	s := SizingPercent(150)
	if s.Value != 100 {
		t.Errorf("SizingPercent should clamp to 100, got %d", s.Value)
	}
	s = SizingPercent(-5)
	if s.Value != 0 {
		t.Errorf("SizingPercent should clamp to 0, got %d", s.Value)
	}
}

func TestSizingFixedNegative(t *testing.T) {
	s := SizingFixed(-10)
	if s.Value != 0 {
		t.Errorf("SizingFixed should clamp negative to 0, got %d", s.Value)
	}
}

func TestCustomBorderChar(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "A", Sizing: SizingFill()},
			{Title: "B", Sizing: SizingFill()},
		},
		ShowHeader: true,
		ShowBorder: true,
		BorderChar: "|",
	}
	dt := NewDataTable(cfg)
	dt.SetRows([]Row{{ID: "1", Cells: []string{"X", "Y"}}})
	out := dt.Render(40, 5)
	if !containsVisible(out, "|") {
		t.Error("should use custom border char |")
	}
}

func TestFilterResetsScrollAndSelection(t *testing.T) {
	cfg := defaultCfg()
	cfg.Selectable = true
	dt := NewDataTable(cfg)
	dt.SetRows(sampleRows())
	dt.ScrollDown(5)
	dt.SelectNext()
	dt.SelectNext()

	dt.SetFilter(func(r Row) bool { return true })
	if dt.scrollOffset != 0 {
		t.Error("SetFilter should reset scrollOffset to 0")
	}
	if dt.selectedIdx != -1 {
		t.Error("SetFilter should reset selectedIdx to -1")
	}
}

func TestRowMissingCells(t *testing.T) {
	cfg := defaultCfg()
	dt := NewDataTable(cfg)
	// Row with fewer cells than columns.
	dt.SetRows([]Row{{ID: "1", Cells: []string{"OnlyOne"}}})
	out := dt.Render(40, 5)
	if !containsVisible(out, "OnlyOne") {
		t.Error("should render available cells even if fewer than columns")
	}
	// Should not panic.
}

func TestLargeDatasetScroll(t *testing.T) {
	cfg := DataTableConfig{
		Columns: []Column{
			{Title: "ID", Sizing: SizingFixed(5), Align: ColAlignRight},
			{Title: "Value", Sizing: SizingFill()},
		},
		ShowHeader: true,
		ShowBorder: true,
	}
	dt := NewDataTable(cfg)
	rows := make([]Row, 1000)
	for i := range rows {
		rows[i] = Row{
			ID:    fmt.Sprintf("%d", i),
			Cells: []string{fmt.Sprintf("%d", i), fmt.Sprintf("Value-%d", i)},
		}
	}
	dt.SetRows(rows)

	// Scroll to middle.
	dt.ScrollDown(500)
	out := dt.Render(40, 10)
	if lineCount(out) != 10 {
		t.Errorf("expected 10 lines, got %d", lineCount(out))
	}
	if !containsVisible(out, "▲") {
		t.Error("should show top indicator when scrolled to middle")
	}
	if !containsVisible(out, "▼") {
		t.Error("should show bottom indicator when scrolled to middle")
	}
}
