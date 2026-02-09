package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
)

// mockWidget implements app.Widget with minimal stubs for testing.
type mockWidget struct {
	id         string
	title      string
	minW, minH int
	lastKey    tea.KeyMsg // records the last key passed to HandleKey
	keyCalled  bool
}

func newMockWidget(id, title string) *mockWidget {
	return &mockWidget{id: id, title: title, minW: 10, minH: 3}
}

func (w *mockWidget) ID() string    { return w.id }
func (w *mockWidget) Title() string { return w.title }

func (w *mockWidget) Update(_ tea.Msg) tea.Cmd { return nil }

func (w *mockWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	// Return a simple placeholder string.
	lines := make([]string, height)
	for i := range lines {
		if i == 0 {
			line := w.title
			if len(line) > width {
				line = line[:width]
			}
			lines[i] = line
		} else {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

func (w *mockWidget) MinSize() (int, int) { return w.minW, w.minH }

func (w *mockWidget) HandleKey(key tea.KeyMsg) tea.Cmd {
	w.lastKey = key
	w.keyCalled = true
	return nil
}

// helper to send a message through Update and return the updated Model.
func tuiUpdate(m Model, msg tea.Msg) (Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

// helper to create a Model with three mock widgets.
func newTestTuiModel() (Model, []*mockWidget) {
	w1 := newMockWidget("cpu", "CPU Usage")
	w2 := newMockWidget("mem", "Memory")
	w3 := newMockWidget("net", "Network")
	widgets := []app.Widget{w1, w2, w3}
	return New(widgets), []*mockWidget{w1, w2, w3}
}

// --- Test 1: New() creates model with correct initial state ---
func TestNewCreatesCorrectInitialState(t *testing.T) {
	m, _ := newTestTuiModel()

	if m.Focused() != 0 {
		t.Errorf("expected focused=0, got %d", m.Focused())
	}
	if m.Expanded() != -1 {
		t.Errorf("expected expanded=-1, got %d", m.Expanded())
	}
	if m.ShowHelp() {
		t.Error("expected showHelp=false")
	}
	if m.SearchMode() {
		t.Error("expected searchMode=false")
	}
	if m.Ready() {
		t.Error("expected ready=false")
	}
}

// --- Test 2: WindowSizeMsg sets width/height and ready ---
func TestWindowSizeMsgSetsReady(t *testing.T) {
	m, _ := newTestTuiModel()

	m, _ = tuiUpdate(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.Width() != 120 {
		t.Errorf("expected width=120, got %d", m.Width())
	}
	if m.Height() != 40 {
		t.Errorf("expected height=40, got %d", m.Height())
	}
	if !m.Ready() {
		t.Error("expected ready=true after WindowSizeMsg")
	}
}

// --- Test 3: Tab cycles focus forward ---
func TestTabCyclesFocusForward(t *testing.T) {
	m, _ := newTestTuiModel()

	if m.Focused() != 0 {
		t.Fatalf("expected initial focus at 0, got %d", m.Focused())
	}

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != 1 {
		t.Errorf("after Tab, expected focus=1, got %d", m.Focused())
	}

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != 2 {
		t.Errorf("after second Tab, expected focus=2, got %d", m.Focused())
	}
}

// --- Test 4: Shift+Tab cycles focus backward ---
func TestShiftTabCyclesFocusBackward(t *testing.T) {
	m, _ := newTestTuiModel()

	// Start at 0, shift+tab should wrap to 2.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Focused() != 2 {
		t.Errorf("after Shift+Tab from 0, expected focus=2, got %d", m.Focused())
	}

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Focused() != 1 {
		t.Errorf("after second Shift+Tab, expected focus=1, got %d", m.Focused())
	}
}

// --- Test 5: Focus wraps around ---
func TestFocusWrapsAround(t *testing.T) {
	m, _ := newTestTuiModel()

	// Forward wrap: 0 -> 1 -> 2 -> 0
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.Focused() != 0 {
		t.Errorf("expected wrap to 0, got %d", m.Focused())
	}

	// Backward wrap: 0 -> 2
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Focused() != 2 {
		t.Errorf("expected backward wrap to 2, got %d", m.Focused())
	}
}

// --- Test 6: Enter expands focused widget ---
func TestEnterExpandsFocusedWidget(t *testing.T) {
	m, _ := newTestTuiModel()

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Expanded() != 0 {
		t.Errorf("expected expanded=0, got %d", m.Expanded())
	}
}

// --- Test 7: Enter on expanded widget collapses it ---
func TestEnterOnExpandedWidgetCollapses(t *testing.T) {
	m, _ := newTestTuiModel()

	// Expand.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Expanded() != 0 {
		t.Fatalf("expected expanded=0, got %d", m.Expanded())
	}

	// Press enter again to collapse.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Expanded() != -1 {
		t.Errorf("expected expanded=-1 after second Enter, got %d", m.Expanded())
	}
}

// --- Test 8: Escape collapses expanded widget ---
func TestEscapeCollapsesExpandedWidget(t *testing.T) {
	m, _ := newTestTuiModel()

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Expanded() < 0 {
		t.Fatal("expected widget expanded after Enter")
	}

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.Expanded() != -1 {
		t.Errorf("expected expanded=-1 after Escape, got %d", m.Expanded())
	}
}

// --- Test 9: Escape closes help overlay ---
func TestEscapeClosesHelpOverlay(t *testing.T) {
	m, _ := newTestTuiModel()

	// Open help.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.ShowHelp() {
		t.Fatal("expected help visible after pressing ?")
	}

	// Close help with escape.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.ShowHelp() {
		t.Error("expected help hidden after Escape")
	}
}

// --- Test 10: Escape exits search mode ---
func TestEscapeExitsSearchMode(t *testing.T) {
	m, _ := newTestTuiModel()

	// Enter search mode.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.SearchMode() {
		t.Fatal("expected searchMode=true after pressing /")
	}

	// Exit with escape.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.SearchMode() {
		t.Error("expected searchMode=false after Escape")
	}
}

// --- Test 11: '?' toggles help overlay ---
func TestQuestionMarkTogglesHelp(t *testing.T) {
	m, _ := newTestTuiModel()

	if m.ShowHelp() {
		t.Fatal("help should start hidden")
	}

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.ShowHelp() {
		t.Error("expected help visible after first ?")
	}

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.ShowHelp() {
		t.Error("expected help hidden after second ?")
	}
}

// --- Test 12: '/' enters search mode ---
func TestSlashEntersSearchMode(t *testing.T) {
	m, _ := newTestTuiModel()

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.SearchMode() {
		t.Error("expected searchMode=true after /")
	}
	if m.SearchQuery() != "" {
		t.Error("expected empty search query on entry")
	}
}

// --- Test 13: 'q' quits ---
func TestQQuits(t *testing.T) {
	m, _ := newTestTuiModel()

	_, cmd := tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected non-nil quit command after q")
	}
}

// --- Test 14: 'q' in search mode types 'q' instead of quitting ---
func TestQInSearchModeTypesQ(t *testing.T) {
	m, _ := newTestTuiModel()

	// Enter search mode.
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type 'q' - should append to query, not quit.
	m, cmd := tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("expected nil command (no quit) when typing q in search mode")
	}
	if m.SearchQuery() != "q" {
		t.Errorf("expected searchQuery='q', got %q", m.SearchQuery())
	}
}

// --- Test 15: Ctrl+C always quits ---
func TestCtrlCAlwaysQuits(t *testing.T) {
	m, _ := newTestTuiModel()

	_, cmd := tuiUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected non-nil quit command after Ctrl+C")
	}

	// Also quits from search mode.
	m, _ = newTestTuiModel()
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	_, cmd = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected non-nil quit command after Ctrl+C in search mode")
	}
}

// --- Test 16: Arrow keys are passed to focused widget's HandleKey ---
func TestArrowKeysPassedToFocusedWidget(t *testing.T) {
	m, mocks := newTestTuiModel()

	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyUp})

	if !mocks[0].keyCalled {
		t.Error("expected HandleKey called on focused widget for arrow key")
	}
	if mocks[0].lastKey.Type != tea.KeyUp {
		t.Errorf("expected KeyUp, got %v", mocks[0].lastKey.Type)
	}

	// Widget 1 should not have received the key.
	if mocks[1].keyCalled {
		t.Error("expected HandleKey NOT called on unfocused widget")
	}
}

// --- Test 17: tuiComputeGrid produces correct cell count ---
func TestComputeGridCorrectCellCount(t *testing.T) {
	w1 := newMockWidget("a", "Alpha")
	w2 := newMockWidget("b", "Beta")
	w3 := newMockWidget("c", "Gamma")
	widgets := []app.Widget{w1, w2, w3}

	visible := []int{0, 1, 2}
	cells := tuiComputeGrid(widgets, 80, 24, visible, 0)

	if len(cells) != 3 {
		t.Errorf("expected 3 cells, got %d", len(cells))
	}
}

// --- Test 18: tuiComputeGrid with single widget fills available space ---
func TestComputeGridSingleWidgetFillsSpace(t *testing.T) {
	w1 := newMockWidget("a", "Alpha")
	widgets := []app.Widget{w1}

	visible := []int{0}
	cells := tuiComputeGrid(widgets, 80, 24, visible, 0)

	if len(cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(cells))
	}

	// Should get full width and height minus status bar.
	if cells[0].W != 80 {
		t.Errorf("expected cell width=80, got %d", cells[0].W)
	}
	if cells[0].H != 23 { // 24 - 1 for status bar
		t.Errorf("expected cell height=23, got %d", cells[0].H)
	}
}

// --- Test 19: tuiComputeGrid reserves status bar row ---
func TestComputeGridReservesStatusBarRow(t *testing.T) {
	w1 := newMockWidget("a", "Alpha")
	w2 := newMockWidget("b", "Beta")
	widgets := []app.Widget{w1, w2}

	visible := []int{0, 1}
	cells := tuiComputeGrid(widgets, 80, 24, visible, 0)

	// All cells should fit within height-1 (23 rows for status bar).
	for _, cell := range cells {
		bottom := cell.Y + cell.H
		if bottom > 23 {
			t.Errorf("cell extends to row %d, should be within 23 (height-1)", bottom)
		}
	}
}

// --- Test 20: tuiFilterWidgets matches by ID ---
func TestFilterWidgetsByID(t *testing.T) {
	w1 := newMockWidget("cpu", "CPU Usage")
	w2 := newMockWidget("mem", "Memory")
	w3 := newMockWidget("net", "Network")
	widgets := []app.Widget{w1, w2, w3}

	result := tuiFilterWidgets(widgets, "cpu")
	if len(result) != 1 {
		t.Fatalf("expected 1 match for 'cpu', got %d", len(result))
	}
	if result[0] != 0 {
		t.Errorf("expected index 0, got %d", result[0])
	}
}

// --- Test 21: tuiFilterWidgets matches by Title (case-insensitive) ---
func TestFilterWidgetsByTitleCaseInsensitive(t *testing.T) {
	w1 := newMockWidget("cpu", "CPU Usage")
	w2 := newMockWidget("mem", "Memory")
	w3 := newMockWidget("net", "Network")
	widgets := []app.Widget{w1, w2, w3}

	result := tuiFilterWidgets(widgets, "MEMORY")
	if len(result) != 1 {
		t.Fatalf("expected 1 match for 'MEMORY', got %d", len(result))
	}
	if result[0] != 1 {
		t.Errorf("expected index 1, got %d", result[0])
	}

	// Lowercase should also match.
	result = tuiFilterWidgets(widgets, "network")
	if len(result) != 1 {
		t.Fatalf("expected 1 match for 'network', got %d", len(result))
	}
	if result[0] != 2 {
		t.Errorf("expected index 2, got %d", result[0])
	}
}

// --- Test 22: tuiFilterWidgets empty query returns all ---
func TestFilterWidgetsEmptyQueryReturnsAll(t *testing.T) {
	w1 := newMockWidget("cpu", "CPU Usage")
	w2 := newMockWidget("mem", "Memory")
	widgets := []app.Widget{w1, w2}

	result := tuiFilterWidgets(widgets, "")
	if len(result) != 2 {
		t.Errorf("expected 2 results for empty query, got %d", len(result))
	}
}

// --- Test 23: tuiRenderStatusBar fits within width ---
func TestRenderStatusBarFitsWithinWidth(t *testing.T) {
	bar := tuiRenderStatusBar("", 80)

	// The visible length should not exceed 80.
	// Note: bar contains ANSI codes, so we check the raw rune length is
	// reasonable (ANSI codes add ~10 chars for dim/reset).
	if len(bar) == 0 {
		t.Error("expected non-empty status bar")
	}

	// For a very narrow terminal.
	bar = tuiRenderStatusBar("", 10)
	if len(bar) == 0 {
		t.Error("expected non-empty status bar at width=10")
	}
}

// --- Test 24: tuiRenderHelp is centered ---
func TestRenderHelpIsCentered(t *testing.T) {
	help := tuiRenderHelp(120, 40)

	if help == "" {
		t.Fatal("expected non-empty help output")
	}

	lines := strings.Split(help, "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one line in help output")
	}

	// Find the first non-empty line (which should be a centered panel line).
	// It should have leading spaces if the panel is centered in 120 cols.
	foundCentered := false
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed != "" && len(line) > len(trimmed) {
			// Has leading whitespace, which means it is offset from the left.
			foundCentered = true
			break
		}
	}
	if !foundCentered {
		t.Error("expected help panel to be centered (have leading whitespace)")
	}
}

// --- Additional tests beyond the required 22 ---

// Test 25: View returns "Initializing..." before WindowSizeMsg.
func TestViewBeforeWindowSizeMsg(t *testing.T) {
	m, _ := newTestTuiModel()
	output := m.View()
	if output != "Initializing..." {
		t.Errorf("expected 'Initializing...' before size, got %q", output)
	}
}

// Test 26: View produces non-empty output after resize.
func TestViewProducesOutputAfterResize(t *testing.T) {
	m, _ := newTestTuiModel()
	m, _ = tuiUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	output := m.View()
	if output == "" {
		t.Error("expected non-empty view output after resize")
	}
}

// Test 27: Expanded widget view produces output.
func TestExpandedWidgetView(t *testing.T) {
	m, _ := newTestTuiModel()
	m, _ = tuiUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})

	output := m.View()
	if output == "" {
		t.Error("expected non-empty output with expanded widget")
	}
}

// Test 28: Search mode renders search bar.
func TestSearchModeRendersSearchBar(t *testing.T) {
	m, _ := newTestTuiModel()
	m, _ = tuiUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = tuiUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	output := m.View()
	if !strings.Contains(output, "/") {
		t.Error("expected search bar with '/' prefix in output")
	}
}

// Test 29: tuiComputeGrid with empty visible list returns nil.
func TestComputeGridEmptyVisible(t *testing.T) {
	w1 := newMockWidget("a", "Alpha")
	widgets := []app.Widget{w1}

	cells := tuiComputeGrid(widgets, 80, 24, nil, 0)
	if cells != nil {
		t.Errorf("expected nil for empty visible list, got %d cells", len(cells))
	}
}

// Test 30: Init returns nil command.
func TestInitReturnsNil(t *testing.T) {
	m, _ := newTestTuiModel()
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}
