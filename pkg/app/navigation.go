package app

// CycleFocusForward moves focus to the next widget in the order list,
// wrapping around to the first widget after the last.
func (m *AppModel) CycleFocusForward() {
	if len(m.widgetOrder) == 0 {
		return
	}

	idx := m.focusedIndex()
	idx = (idx + 1) % len(m.widgetOrder)
	m.focusedWidget = m.widgetOrder[idx]
}

// CycleFocusBackward moves focus to the previous widget in the order list,
// wrapping around to the last widget before the first.
func (m *AppModel) CycleFocusBackward() {
	if len(m.widgetOrder) == 0 {
		return
	}

	idx := m.focusedIndex()
	idx = (idx - 1 + len(m.widgetOrder)) % len(m.widgetOrder)
	m.focusedWidget = m.widgetOrder[idx]
}

// FocusWidget directly sets focus to the widget with the given ID.
// If the ID is not found in the registry, focus does not change.
func (m *AppModel) FocusWidget(id string) {
	if _, ok := m.widgets[id]; ok {
		m.focusedWidget = id
	}
}

// ToggleExpand toggles the currently focused widget between normal and
// fullscreen mode. If a different widget is already expanded, it switches
// expansion to the focused widget.
func (m *AppModel) ToggleExpand() {
	if m.focusedWidget == "" {
		return
	}

	if m.expandedWidget == m.focusedWidget {
		m.expandedWidget = ""
	} else {
		m.expandedWidget = m.focusedWidget
	}
}

// focusedIndex returns the index of the currently focused widget in the
// order list. Returns 0 if not found.
func (m *AppModel) focusedIndex() int {
	for i, id := range m.widgetOrder {
		if id == m.focusedWidget {
			return i
		}
	}
	return 0
}
