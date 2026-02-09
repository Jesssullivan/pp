package starship

// Config controls which segments appear in the starship output.
type Config struct {
	ShowClaude    bool
	ShowBilling   bool
	ShowTailscale bool
	ShowK8s       bool
	ShowSystem    bool
	CacheDir      string // where to read cached collector data
	MaxWidth      int    // max visible width (default 60)
}

// Segment represents a single piece of the status line.
type Segment struct {
	Icon  string // emoji or nerd font icon
	Text  string // the actual content
	Color string // ANSI color code
}

// ssDefaultMaxWidth is the default maximum visible character width for the
// starship output line.
const ssDefaultMaxWidth = 60

// Render reads cached data and produces a single-line starship module string.
// Returns an empty string if no data is available (starship hides empty
// modules).
func Render(cfg Config) string {
	maxWidth := cfg.MaxWidth
	if maxWidth <= 0 {
		maxWidth = ssDefaultMaxWidth
	}

	var segments []*Segment

	if cfg.ShowClaude {
		if seg := ssClaudeSegment(cfg.CacheDir); seg != nil {
			segments = append(segments, seg)
		}
	}

	if cfg.ShowBilling {
		if seg := ssBillingSegment(cfg.CacheDir); seg != nil {
			segments = append(segments, seg)
		}
	}

	if cfg.ShowTailscale {
		if seg := ssTailscaleSegment(cfg.CacheDir); seg != nil {
			segments = append(segments, seg)
		}
	}

	if cfg.ShowK8s {
		if seg := ssK8sSegment(cfg.CacheDir); seg != nil {
			segments = append(segments, seg)
		}
	}

	if cfg.ShowSystem {
		if seg := ssSystemSegment(cfg.CacheDir); seg != nil {
			segments = append(segments, seg)
		}
	}

	return ssFormatLine(segments, maxWidth)
}
