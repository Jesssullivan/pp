package theme

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/BurntSushi/toml"
)

// thTOMLTheme is the TOML-serializable representation of a Theme.
type thTOMLTheme struct {
	Name    string        `toml:"name"`
	Base    thTOMLBase    `toml:"base"`
	Widget  thTOMLWidget  `toml:"widget"`
	Status  thTOMLStatus  `toml:"status"`
	Gauge   thTOMLGauge   `toml:"gauge"`
	Chart   thTOMLChart   `toml:"chart"`
	Special thTOMLSpecial `toml:"special"`
}

type thTOMLBase struct {
	Background string `toml:"background"`
	Foreground string `toml:"foreground"`
	Dim        string `toml:"dim"`
	Accent     string `toml:"accent"`
}

type thTOMLWidget struct {
	Border      string `toml:"border"`
	BorderFocus string `toml:"border_focus"`
	Title       string `toml:"title"`
}

type thTOMLStatus struct {
	OK      string `toml:"ok"`
	Warn    string `toml:"warn"`
	Error   string `toml:"error"`
	Unknown string `toml:"unknown"`
}

type thTOMLGauge struct {
	Filled string `toml:"filled"`
	Empty  string `toml:"empty"`
	Warn   string `toml:"warn"`
	Crit   string `toml:"crit"`
}

type thTOMLChart struct {
	Line string `toml:"line"`
	Fill string `toml:"fill"`
	Grid string `toml:"grid"`
}

type thTOMLSpecial struct {
	SearchHighlight string `toml:"search_highlight"`
	HelpKey         string `toml:"help_key"`
	HelpDesc        string `toml:"help_desc"`
}

var thHexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// LoadFromTOML parses a TOML theme definition from raw bytes.
func LoadFromTOML(data []byte) (Theme, error) {
	var tt thTOMLTheme
	if err := toml.Unmarshal(data, &tt); err != nil {
		return Theme{}, fmt.Errorf("theme: parse TOML: %w", err)
	}

	t := Theme{
		Name:       tt.Name,
		Background: tt.Base.Background,
		Foreground: tt.Base.Foreground,
		Dim:        tt.Base.Dim,
		Accent:     tt.Base.Accent,

		Border:      tt.Widget.Border,
		BorderFocus: tt.Widget.BorderFocus,
		Title:       tt.Widget.Title,

		StatusOK:      tt.Status.OK,
		StatusWarn:    tt.Status.Warn,
		StatusError:   tt.Status.Error,
		StatusUnknown: tt.Status.Unknown,

		GaugeFilled: tt.Gauge.Filled,
		GaugeEmpty:  tt.Gauge.Empty,
		GaugeWarn:   tt.Gauge.Warn,
		GaugeCrit:   tt.Gauge.Crit,

		ChartLine: tt.Chart.Line,
		ChartFill: tt.Chart.Fill,
		ChartGrid: tt.Chart.Grid,

		SearchHighlight: tt.Special.SearchHighlight,
		HelpKey:         tt.Special.HelpKey,
		HelpDesc:        tt.Special.HelpDesc,
	}

	if err := thValidateTheme(t); err != nil {
		return Theme{}, err
	}

	return t, nil
}

// SaveToTOML serializes a theme to TOML bytes.
func SaveToTOML(t Theme) ([]byte, error) {
	tt := thTOMLTheme{
		Name: t.Name,
		Base: thTOMLBase{
			Background: t.Background,
			Foreground: t.Foreground,
			Dim:        t.Dim,
			Accent:     t.Accent,
		},
		Widget: thTOMLWidget{
			Border:      t.Border,
			BorderFocus: t.BorderFocus,
			Title:       t.Title,
		},
		Status: thTOMLStatus{
			OK:      t.StatusOK,
			Warn:    t.StatusWarn,
			Error:   t.StatusError,
			Unknown: t.StatusUnknown,
		},
		Gauge: thTOMLGauge{
			Filled: t.GaugeFilled,
			Empty:  t.GaugeEmpty,
			Warn:   t.GaugeWarn,
			Crit:   t.GaugeCrit,
		},
		Chart: thTOMLChart{
			Line: t.ChartLine,
			Fill: t.ChartFill,
			Grid: t.ChartGrid,
		},
		Special: thTOMLSpecial{
			SearchHighlight: t.SearchHighlight,
			HelpKey:         t.HelpKey,
			HelpDesc:        t.HelpDesc,
		},
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(tt); err != nil {
		return nil, fmt.Errorf("theme: encode TOML: %w", err)
	}
	return buf.Bytes(), nil
}

// thValidateTheme checks that all required color fields are present and valid hex.
func thValidateTheme(t Theme) error {
	fields := map[string]string{
		"name":             t.Name,
		"background":       t.Background,
		"foreground":       t.Foreground,
		"dim":              t.Dim,
		"accent":           t.Accent,
		"border":           t.Border,
		"border_focus":     t.BorderFocus,
		"title":            t.Title,
		"status_ok":        t.StatusOK,
		"status_warn":      t.StatusWarn,
		"status_error":     t.StatusError,
		"status_unknown":   t.StatusUnknown,
		"gauge_filled":     t.GaugeFilled,
		"gauge_empty":      t.GaugeEmpty,
		"gauge_warn":       t.GaugeWarn,
		"gauge_crit":       t.GaugeCrit,
		"chart_line":       t.ChartLine,
		"chart_fill":       t.ChartFill,
		"chart_grid":       t.ChartGrid,
		"search_highlight": t.SearchHighlight,
		"help_key":         t.HelpKey,
		"help_desc":        t.HelpDesc,
	}

	for field, value := range fields {
		if value == "" {
			return fmt.Errorf("theme: missing required field %q", field)
		}
	}

	// Validate hex colors (all fields except "name").
	colorFields := map[string]string{
		"background":       t.Background,
		"foreground":       t.Foreground,
		"dim":              t.Dim,
		"accent":           t.Accent,
		"border":           t.Border,
		"border_focus":     t.BorderFocus,
		"title":            t.Title,
		"status_ok":        t.StatusOK,
		"status_warn":      t.StatusWarn,
		"status_error":     t.StatusError,
		"status_unknown":   t.StatusUnknown,
		"gauge_filled":     t.GaugeFilled,
		"gauge_empty":      t.GaugeEmpty,
		"gauge_warn":       t.GaugeWarn,
		"gauge_crit":       t.GaugeCrit,
		"chart_line":       t.ChartLine,
		"chart_fill":       t.ChartFill,
		"chart_grid":       t.ChartGrid,
		"search_highlight": t.SearchHighlight,
		"help_key":         t.HelpKey,
		"help_desc":        t.HelpDesc,
	}

	for field, value := range colorFields {
		if !thHexColorRegex.MatchString(value) {
			return fmt.Errorf("theme: invalid hex color %q for field %q (expected #RRGGBB)", value, field)
		}
	}

	return nil
}
