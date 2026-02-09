package theme

import (
	"sort"
	"strings"
	"sync"
)

// Theme defines the complete color palette for the dashboard.
type Theme struct {
	Name string

	// Base colors
	Background string // hex color e.g. "#1a1b26"
	Foreground string // hex color
	Dim        string // dimmed text
	Accent     string // highlights, focused borders

	// Widget colors
	Border      string // unfocused widget borders
	BorderFocus string // focused widget border
	Title       string // widget title text

	// Status colors
	StatusOK      string // green - healthy
	StatusWarn    string // yellow - warning
	StatusError   string // red - error
	StatusUnknown string // gray - unknown

	// Gauge colors
	GaugeFilled string
	GaugeEmpty  string
	GaugeWarn   string
	GaugeCrit   string

	// Chart colors (for sparklines, timegraphs)
	ChartLine string
	ChartFill string
	ChartGrid string

	// Special
	SearchHighlight string
	HelpKey         string // keybinding highlight color
	HelpDesc        string // help description color
}

// Current holds the active theme (set via SetCurrent).
var Current Theme

var (
	mu       sync.RWMutex
	registry = map[string]Theme{}
)

func init() {
	thRegisterBuiltins()
	Current = thDefaultTheme()
}

// Get returns a named theme, falling back to Default if not found.
func Get(name string) Theme {
	mu.RLock()
	defer mu.RUnlock()
	if t, ok := registry[strings.ToLower(name)]; ok {
		return t
	}
	return registry["default"]
}

// Names returns all available theme names sorted alphabetically.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetCurrent sets the active theme by name.
func SetCurrent(name string) {
	Current = Get(name)
}

// thRegister adds a theme to the registry under its lowercase name.
func thRegister(t Theme) {
	mu.Lock()
	defer mu.Unlock()
	registry[strings.ToLower(t.Name)] = t
}
