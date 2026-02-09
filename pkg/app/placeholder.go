package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PlaceholderWidget is a minimal widget that displays its ID and the
// dimensions it was rendered at. It is used for testing layout, navigation,
// and the widget interface before real widgets are built.
type PlaceholderWidget struct {
	id    string
	title string
}

// NewPlaceholder creates a new PlaceholderWidget with the given id and title.
func NewPlaceholder(id, title string) *PlaceholderWidget {
	return &PlaceholderWidget{id: id, title: title}
}

// ID returns the widget's unique identifier.
func (w *PlaceholderWidget) ID() string {
	return w.id
}

// Title returns the widget's display title.
func (w *PlaceholderWidget) Title() string {
	return w.title
}

// Update is a no-op for the placeholder widget.
func (w *PlaceholderWidget) Update(_ tea.Msg) tea.Cmd {
	return nil
}

// View renders a simple box showing the widget's title and the dimensions
// it was asked to render at.
func (w *PlaceholderWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	titleLine := titleStyle.Render(w.title)
	dimLine := dimStyle.Render(fmt.Sprintf("%dx%d", width, height))

	// Center the content vertically within the available height.
	var lines []string

	// Pad top.
	topPad := (height - 2) / 2
	if topPad < 0 {
		topPad = 0
	}
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}

	lines = append(lines, titleLine)
	if height > 1 {
		lines = append(lines, dimLine)
	}

	// Pad bottom to fill height.
	for len(lines) < height {
		lines = append(lines, "")
	}

	// Truncate if we somehow exceed height.
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// MinSize returns the minimum dimensions for the placeholder widget.
func (w *PlaceholderWidget) MinSize() (int, int) {
	return 10, 3
}

// HandleKey is a no-op for the placeholder widget.
func (w *PlaceholderWidget) HandleKey(_ tea.KeyMsg) tea.Cmd {
	return nil
}
