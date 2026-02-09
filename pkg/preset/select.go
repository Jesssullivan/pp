package preset

import (
	"sort"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
)

// Terminal width thresholds for auto-selection.
const (
	prSmallMaxCols = 100
	prLargeMinCols = 160
)

// SelectForSize auto-selects a preset name based on terminal dimensions.
//   - Small (<100 cols): "minimal"
//   - Medium (100-160): "dashboard"
//   - Large (>160): "dashboard"
func SelectForSize(width, height int) string {
	_ = height // reserved for future row-based selection
	if width < prSmallMaxCols {
		return "minimal"
	}
	return "dashboard"
}

// SelectByConfig reads the preset name from config. If the preset is "auto",
// it triggers size-based selection using the provided terminal dimensions.
// The width and height should be the current terminal size; they are only
// used when the config preset is "auto".
func SelectByConfig(cfg config.Config) string {
	name := cfg.Layout.Preset
	if name == "" || name == "auto" {
		// Without terminal dimensions available from config alone,
		// return "dashboard" as the safe default. Callers that need
		// size-aware selection should use SelectForSize directly.
		return "dashboard"
	}
	return name
}

// prFilterByPriority keeps only the top maxSlots widgets sorted by priority
// (highest first). The returned slice preserves the original order of kept
// widgets for stable layout positioning.
func prFilterByPriority(slots []WidgetSlot, maxSlots int) []WidgetSlot {
	if maxSlots <= 0 {
		return nil
	}
	if len(slots) <= maxSlots {
		result := make([]WidgetSlot, len(slots))
		copy(result, slots)
		return result
	}

	// Build index-priority pairs, sort by descending priority.
	type indexedSlot struct {
		index int
		slot  WidgetSlot
	}
	indexed := make([]indexedSlot, len(slots))
	for i, s := range slots {
		indexed[i] = indexedSlot{index: i, slot: s}
	}
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].slot.Priority > indexed[j].slot.Priority
	})

	// Keep top maxSlots by priority.
	kept := indexed[:maxSlots]

	// Re-sort by original index for stable ordering.
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].index < kept[j].index
	})

	result := make([]WidgetSlot, maxSlots)
	for i, k := range kept {
		result[i] = k.slot
	}
	return result
}
