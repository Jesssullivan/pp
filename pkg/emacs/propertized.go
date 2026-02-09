package emacs

import (
	"fmt"
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/cache"
)

// RenderPropertized produces text with Emacs text property annotations.
// Format: #("text" start end (face face-name)) for each styled span.
// This is a simplified propertized string that Emacs can read with `read`.
func RenderPropertized(cacheDir string) (string, error) {
	store, err := emOpenStore(cacheDir)
	if err != nil {
		return "", fmt.Errorf("emacs propertized: open cache: %w", err)
	}
	defer store.Close()

	var sections []string

	extractors := []func(s *cache.Store) *WidgetJSON{
		emExtractClaude,
		emExtractBilling,
		emExtractTailscale,
		emExtractK8s,
		emExtractSystem,
	}

	for _, extract := range extractors {
		w := extract(store)
		if w == nil {
			continue
		}

		// Section header with bold face.
		header := emPropertize(w.Title, "bold")

		// Status indicator with appropriate face.
		face := emStatusFace(w.Status)
		indicator := emPropertize(emStatusIcon(w.Status), face)

		// Summary with keyword face.
		summary := emPropertize(w.Summary, "font-lock-string-face")

		sections = append(sections, fmt.Sprintf("%s %s %s", header, indicator, summary))
	}

	if len(sections) == 0 {
		return emPropertize("No data available", "font-lock-comment-face"), nil
	}

	return strings.Join(sections, "\n"), nil
}

// emPropertize wraps text in Emacs propertized string format.
// Format: #("text" 0 len (face face-name))
func emPropertize(text, face string) string {
	if text == "" {
		return "#(\"\" 0 0 (face " + face + "))"
	}
	return fmt.Sprintf("#(%q 0 %d (face %s))", text, len(text), face)
}

// emStatusFace returns the Emacs face name for a widget status.
func emStatusFace(status string) string {
	switch status {
	case "ok":
		return "success"
	case "warning":
		return "font-lock-warning-face"
	case "error":
		return "error"
	default:
		return "font-lock-keyword-face"
	}
}

// emStatusIcon returns a text indicator for a widget status.
func emStatusIcon(status string) string {
	switch status {
	case "ok":
		return "[OK]"
	case "warning":
		return "[WARN]"
	case "error":
		return "[ERR]"
	default:
		return "[??]"
	}
}
