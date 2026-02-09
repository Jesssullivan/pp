package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// helper to create a model with 3 placeholder widgets for testing.
func newTestModel() AppModel {
	cfg := DefaultConfig()
	return NewAppModel(cfg,
		NewPlaceholder("cpu", "CPU"),
		NewPlaceholder("mem", "Memory"),
		NewPlaceholder("net", "Network"),
	)
}

// helper to send a message through Update and return the updated model.
func update(m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(AppModel), cmd
}

func TestInitReturnsTickCmd(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, expected a tick command")
	}
	// Execute the command and verify it produces a TickEvent.
	// tea.Tick returns a function that sleeps, so we just verify it is non-nil.
}

func TestWindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := newTestModel()
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.Width() != 120 {
		t.Errorf("expected width 120, got %d", m.Width())
	}
	if m.Height() != 40 {
		t.Errorf("expected height 40, got %d", m.Height())
	}
}

func TestWindowSizeMsgMarksLayoutDirty(t *testing.T) {
	m := newTestModel()
	// Clear the initial dirty flag by computing layout.
	m.layoutDirty = false

	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	if !m.LayoutDirty() {
		t.Error("expected layoutDirty=true after WindowSizeMsg")
	}
}

func TestTabCyclesFocusForward(t *testing.T) {
	m := newTestModel()

	if m.FocusedWidgetID() != "cpu" {
		t.Fatalf("expected initial focus on 'cpu', got %q", m.FocusedWidgetID())
	}

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.FocusedWidgetID() != "mem" {
		t.Errorf("after first Tab, expected focus on 'mem', got %q", m.FocusedWidgetID())
	}

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.FocusedWidgetID() != "net" {
		t.Errorf("after second Tab, expected focus on 'net', got %q", m.FocusedWidgetID())
	}

	// Wrap around.
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.FocusedWidgetID() != "cpu" {
		t.Errorf("after third Tab, expected focus to wrap to 'cpu', got %q", m.FocusedWidgetID())
	}
}

func TestShiftTabCyclesFocusBackward(t *testing.T) {
	m := newTestModel()

	if m.FocusedWidgetID() != "cpu" {
		t.Fatalf("expected initial focus on 'cpu', got %q", m.FocusedWidgetID())
	}

	// Backward from first should wrap to last.
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.FocusedWidgetID() != "net" {
		t.Errorf("after Shift+Tab from 'cpu', expected 'net', got %q", m.FocusedWidgetID())
	}

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.FocusedWidgetID() != "mem" {
		t.Errorf("after second Shift+Tab, expected 'mem', got %q", m.FocusedWidgetID())
	}
}

func TestEnterExpandsFocusedWidget(t *testing.T) {
	m := newTestModel()

	if m.ExpandedWidgetID() != "" {
		t.Fatalf("expected no expanded widget initially, got %q", m.ExpandedWidgetID())
	}

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.ExpandedWidgetID() != "cpu" {
		t.Errorf("after Enter, expected expanded='cpu', got %q", m.ExpandedWidgetID())
	}
}

func TestEscCollapsesExpandedWidget(t *testing.T) {
	m := newTestModel()

	// First expand.
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.ExpandedWidgetID() == "" {
		t.Fatal("widget should be expanded after Enter")
	}

	// Then collapse.
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.ExpandedWidgetID() != "" {
		t.Errorf("after Esc, expected no expanded widget, got %q", m.ExpandedWidgetID())
	}
}

func TestEscNoOpWhenNothingExpanded(t *testing.T) {
	m := newTestModel()

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.ExpandedWidgetID() != "" {
		t.Errorf("Esc with nothing expanded should be no-op, got expanded=%q", m.ExpandedWidgetID())
	}
}

func TestQSendsQuitMessage(t *testing.T) {
	m := newTestModel()

	m, cmd := update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !m.Quitting() {
		t.Error("expected quitting=true after pressing q")
	}

	if cmd == nil {
		t.Error("expected non-nil quit command after pressing q")
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := newTestModel()

	m, cmd := update(m, tea.KeyMsg{Type: tea.KeyCtrlC})

	if !m.Quitting() {
		t.Error("expected quitting=true after Ctrl+C")
	}
	if cmd == nil {
		t.Error("expected non-nil quit command after Ctrl+C")
	}
}

func TestQuestionMarkTogglesHelp(t *testing.T) {
	m := newTestModel()

	if m.HelpVisible() {
		t.Fatal("help should not be visible initially")
	}

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.HelpVisible() {
		t.Error("help should be visible after pressing ?")
	}

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.HelpVisible() {
		t.Error("help should be hidden after pressing ? again")
	}
}

func TestDataUpdateEventStoresData(t *testing.T) {
	m := newTestModel()

	testData := map[string]string{"status": "ok"}
	m, _ = update(m, DataUpdateEvent{
		Source:    "tailscale",
		Data:     testData,
		Timestamp: time.Now(),
	})

	stored, ok := m.DataStore()["tailscale"]
	if !ok {
		t.Fatal("expected 'tailscale' key in dataStore")
	}

	storedMap, ok := stored.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", stored)
	}
	if storedMap["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", storedMap["status"])
	}
}

func TestDataUpdateEventWithErrorDoesNotStore(t *testing.T) {
	m := newTestModel()

	m, _ = update(m, DataUpdateEvent{
		Source:    "failing",
		Data:     nil,
		Err:      &testError{"fetch failed"},
		Timestamp: time.Now(),
	})

	if _, ok := m.DataStore()["failing"]; ok {
		t.Error("expected no data stored for a failed fetch")
	}
}

func TestViewProducesNonEmptyOutputSmallTerminal(t *testing.T) {
	m := newTestModel()
	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	output := m.View()
	if output == "" {
		t.Error("View() at 80x24 should produce non-empty output")
	}
}

func TestViewProducesNonEmptyOutputLargeTerminal(t *testing.T) {
	m := newTestModel()
	m, _ = update(m, tea.WindowSizeMsg{Width: 200, Height: 60})

	output := m.View()
	if output == "" {
		t.Error("View() at 200x60 should produce non-empty output")
	}
}

func TestViewReturnsInitializingBeforeResize(t *testing.T) {
	m := newTestModel()
	output := m.View()
	if output != "Initializing..." {
		t.Errorf("expected 'Initializing...' before WindowSizeMsg, got %q", output)
	}
}

func TestViewReturnsEmptyWhenQuitting(t *testing.T) {
	m := newTestModel()
	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	output := m.View()
	if output != "" {
		t.Errorf("expected empty view when quitting, got %q", output)
	}
}

func TestExpandedWidgetRendersFullscreen(t *testing.T) {
	m := newTestModel()
	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyEnter})

	output := m.View()
	if output == "" {
		t.Error("expected non-empty output with expanded widget")
	}
}

func TestFocusWidgetByID(t *testing.T) {
	m := newTestModel()

	m.FocusWidget("net")
	if m.FocusedWidgetID() != "net" {
		t.Errorf("expected focus on 'net', got %q", m.FocusedWidgetID())
	}
}

func TestFocusWidgetInvalidIDNoOp(t *testing.T) {
	m := newTestModel()

	m.FocusWidget("nonexistent")
	if m.FocusedWidgetID() != "cpu" {
		t.Errorf("expected focus unchanged at 'cpu', got %q", m.FocusedWidgetID())
	}
}

func TestToggleExpandTwiceReturnsToNormal(t *testing.T) {
	m := newTestModel()

	m.ToggleExpand()
	if m.ExpandedWidgetID() == "" {
		t.Fatal("expected widget expanded after first ToggleExpand")
	}

	m.ToggleExpand()
	if m.ExpandedWidgetID() != "" {
		t.Error("expected no expanded widget after second ToggleExpand")
	}
}

func TestNewAppModelWithNoWidgets(t *testing.T) {
	m := NewAppModel(nil)

	if m.FocusedWidgetID() != "" {
		t.Errorf("expected no focused widget with empty model, got %q", m.FocusedWidgetID())
	}

	// Should not panic.
	m.CycleFocusForward()
	m.CycleFocusBackward()
	m.ToggleExpand()
}

func TestTickEventReturnsTickCmd(t *testing.T) {
	m := newTestModel()

	_, cmd := update(m, TickEvent{Time: time.Now()})
	if cmd == nil {
		t.Error("expected TickEvent to return a new tick command")
	}
}

func TestDataFetchCmd(t *testing.T) {
	cmd := DataFetchCmd("test", func() (interface{}, error) {
		return "hello", nil
	})

	if cmd == nil {
		t.Fatal("DataFetchCmd returned nil")
	}

	msg := cmd()
	ev, ok := msg.(DataUpdateEvent)
	if !ok {
		t.Fatalf("expected DataUpdateEvent, got %T", msg)
	}
	if ev.Source != "test" {
		t.Errorf("expected source='test', got %q", ev.Source)
	}
	if ev.Data != "hello" {
		t.Errorf("expected data='hello', got %v", ev.Data)
	}
	if ev.Err != nil {
		t.Errorf("expected no error, got %v", ev.Err)
	}
}

func TestDataFetchCmdWithError(t *testing.T) {
	cmd := DataFetchCmd("failing", func() (interface{}, error) {
		return nil, &testError{"boom"}
	})

	msg := cmd()
	ev := msg.(DataUpdateEvent)
	if ev.Err == nil {
		t.Error("expected error in DataUpdateEvent")
	}
	if ev.Data != nil {
		t.Error("expected nil data when fetch fails")
	}
}

func TestPlaceholderWidgetInterface(t *testing.T) {
	w := NewPlaceholder("test", "Test Widget")

	if w.ID() != "test" {
		t.Errorf("expected ID='test', got %q", w.ID())
	}
	if w.Title() != "Test Widget" {
		t.Errorf("expected Title='Test Widget', got %q", w.Title())
	}

	minW, minH := w.MinSize()
	if minW < 1 || minH < 1 {
		t.Errorf("expected positive MinSize, got %dx%d", minW, minH)
	}

	view := w.View(40, 10)
	if view == "" {
		t.Error("expected non-empty View output")
	}

	// Update and HandleKey should not panic and return nil.
	if cmd := w.Update(nil); cmd != nil {
		t.Error("expected nil from placeholder Update")
	}
	if cmd := w.HandleKey(tea.KeyMsg{}); cmd != nil {
		t.Error("expected nil from placeholder HandleKey")
	}
}

func TestPlaceholderViewZeroDimensions(t *testing.T) {
	w := NewPlaceholder("test", "Test")

	if v := w.View(0, 0); v != "" {
		t.Errorf("expected empty string for 0x0, got %q", v)
	}
	if v := w.View(-1, 10); v != "" {
		t.Errorf("expected empty string for negative width, got %q", v)
	}
}

func TestHelpOverlayInView(t *testing.T) {
	m := newTestModel()
	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Toggle help on.
	m, _ = update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	output := m.View()
	if output == "" {
		t.Error("expected non-empty output with help visible")
	}
}

func TestMultipleDataUpdates(t *testing.T) {
	m := newTestModel()

	m, _ = update(m, DataUpdateEvent{
		Source:    "cpu",
		Data:     42,
		Timestamp: time.Now(),
	})
	m, _ = update(m, DataUpdateEvent{
		Source:    "mem",
		Data:     "8GB",
		Timestamp: time.Now(),
	})

	if m.DataStore()["cpu"] != 42 {
		t.Errorf("expected cpu=42, got %v", m.DataStore()["cpu"])
	}
	if m.DataStore()["mem"] != "8GB" {
		t.Errorf("expected mem='8GB', got %v", m.DataStore()["mem"])
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RefreshInterval <= 0 {
		t.Error("expected positive RefreshInterval in DefaultConfig")
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
