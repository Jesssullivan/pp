package tui

import (
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// tuiRenderSearchBar renders the search input bar that replaces the
// status bar when search mode is active. It shows a "/" prefix,
// the current query, and a cursor indicator.
func tuiRenderSearchBar(query string, width int) string {
	if width <= 0 {
		return ""
	}

	display := "/" + query + "_"

	return components.PadRight(components.Truncate(display, width), width)
}

// tuiFilterWidgets returns the indices of widgets whose ID or Title
// matches the query (case-insensitive substring match). An empty query
// returns all indices.
func tuiFilterWidgets(widgets []app.Widget, query string) []int {
	if query == "" {
		indices := make([]int, len(widgets))
		for i := range widgets {
			indices[i] = i
		}
		return indices
	}

	lower := strings.ToLower(query)
	var result []int

	for i, w := range widgets {
		id := strings.ToLower(w.ID())
		title := strings.ToLower(w.Title())

		if strings.Contains(id, lower) || strings.Contains(title, lower) {
			result = append(result, i)
		}
	}

	return result
}
