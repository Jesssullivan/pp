package preset

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
)

// prTomlPreset is the TOML-friendly representation used for serialization.
type prTomlPreset struct {
	Name        string         `toml:"name"`
	Description string         `toml:"description,omitempty"`
	Columns     int            `toml:"columns"`
	Widgets     []prTomlWidget `toml:"widgets"`
}

// prTomlWidget is the TOML-friendly representation of a WidgetSlot.
type prTomlWidget struct {
	ID       string `toml:"id"`
	Column   int    `toml:"column"`
	Row      int    `toml:"row"`
	ColSpan  int    `toml:"col_span,omitempty"`
	RowSpan  int    `toml:"row_span,omitempty"`
	MinW     int    `toml:"min_w,omitempty"`
	MinH     int    `toml:"min_h,omitempty"`
	Priority int    `toml:"priority,omitempty"`
}

// LoadFromTOML parses a custom layout preset from TOML data.
func LoadFromTOML(data []byte) (LayoutPreset, error) {
	var raw prTomlPreset
	if err := toml.Unmarshal(data, &raw); err != nil {
		return LayoutPreset{}, fmt.Errorf("preset: parse TOML: %w", err)
	}

	if raw.Name == "" {
		return LayoutPreset{}, fmt.Errorf("preset: missing required field 'name'")
	}

	if raw.Columns <= 0 {
		raw.Columns = 1
	}

	widgets := make([]WidgetSlot, len(raw.Widgets))
	for i, w := range raw.Widgets {
		widgets[i] = WidgetSlot{
			WidgetID: w.ID,
			Column:   w.Column,
			Row:      w.Row,
			ColSpan:  w.ColSpan,
			RowSpan:  w.RowSpan,
			MinW:     w.MinW,
			MinH:     w.MinH,
			Priority: w.Priority,
		}
	}

	return LayoutPreset{
		Name:        raw.Name,
		Description: raw.Description,
		Columns:     raw.Columns,
		Widgets:     widgets,
	}, nil
}

// SaveToTOML serializes a preset to TOML format.
func SaveToTOML(preset LayoutPreset) ([]byte, error) {
	raw := prTomlPreset{
		Name:        preset.Name,
		Description: preset.Description,
		Columns:     preset.Columns,
		Widgets:     make([]prTomlWidget, len(preset.Widgets)),
	}

	for i, w := range preset.Widgets {
		raw.Widgets[i] = prTomlWidget{
			ID:       w.WidgetID,
			Column:   w.Column,
			Row:      w.Row,
			ColSpan:  w.ColSpan,
			RowSpan:  w.RowSpan,
			MinW:     w.MinW,
			MinH:     w.MinH,
			Priority: w.Priority,
		}
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(raw); err != nil {
		return nil, fmt.Errorf("preset: encode TOML: %w", err)
	}
	return buf.Bytes(), nil
}
