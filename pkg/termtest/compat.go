package termtest

// CompatResult describes the expected behavior of a feature on a terminal.
type CompatResult struct {
	Feature    string // Feature name from Features()
	Terminal   string // Terminal profile name
	Status     string // "full", "partial", "degraded", "unsupported"
	Notes      string // Human-readable explanation
	Workaround string // If degraded/unsupported, what workaround to use
}

// Features returns all dashboard features that vary by terminal.
func Features() []string {
	return []string{
		"truecolor",
		"image_rendering",
		"unicode_box_drawing",
		"braille_chars",
		"sparkline_chars",
		"mouse_navigation",
		"clipboard_integration",
		"resize_detection",
		"pixel_sizing",
	}
}

// CheckCompat evaluates a terminal profile against all dashboard features.
func CheckCompat(profile TerminalProfile) []CompatResult {
	features := Features()
	results := make([]CompatResult, 0, len(features))
	for _, feat := range features {
		results = append(results, ttCheckFeature(feat, profile))
	}
	return results
}

// ttCheckFeature evaluates a single feature against a terminal profile.
func ttCheckFeature(feature string, profile TerminalProfile) CompatResult {
	r := CompatResult{
		Feature:  feature,
		Terminal: profile.Name,
	}

	switch feature {
	case "truecolor":
		r = ttCheckTruecolor(r, profile)
	case "image_rendering":
		r = ttCheckImageRendering(r, profile)
	case "unicode_box_drawing":
		r = ttCheckBoxDrawing(r, profile)
	case "braille_chars":
		r = ttCheckBraille(r, profile)
	case "sparkline_chars":
		r = ttCheckSparkline(r, profile)
	case "mouse_navigation":
		r = ttCheckMouse(r, profile)
	case "clipboard_integration":
		r = ttCheckClipboard(r, profile)
	case "resize_detection":
		r = ttCheckResize(r, profile)
	case "pixel_sizing":
		r = ttCheckPixelSizing(r, profile)
	default:
		r.Status = "unsupported"
		r.Notes = "unknown feature"
	}

	return r
}

func ttCheckTruecolor(r CompatResult, p TerminalProfile) CompatResult {
	if p.ColorDepth == 24 {
		r.Status = "full"
		r.Notes = "24-bit truecolor supported"
	} else {
		r.Status = "degraded"
		r.Notes = "limited to 256 colors"
		r.Workaround = "use 256-color palette approximation"
	}
	return r
}

func ttCheckImageRendering(r CompatResult, p TerminalProfile) CompatResult {
	switch p.ImageProtocol {
	case "kitty":
		r.Status = "full"
		r.Notes = "kitty graphics protocol for high-quality images"
	case "iterm2":
		r.Status = "full"
		r.Notes = "iTerm2 inline images protocol"
	case "sixel":
		r.Status = "partial"
		r.Notes = "sixel protocol with limited color depth"
		r.Workaround = "dither images to sixel palette"
	case "halfblock":
		r.Status = "degraded"
		r.Notes = "half-block character approximation only"
		r.Workaround = "render images using unicode half-block characters"
	default:
		r.Status = "unsupported"
		r.Notes = "no image rendering protocol available"
		r.Workaround = "skip image rendering entirely"
	}
	return r
}

func ttCheckBoxDrawing(r CompatResult, p TerminalProfile) CompatResult {
	if p.BoxDrawing && p.UnicodeFull {
		r.Status = "full"
		r.Notes = "full unicode box drawing with extended characters"
	} else if p.BoxDrawing {
		r.Status = "partial"
		r.Notes = "basic box drawing characters supported"
		r.Workaround = "use ASCII box drawing fallback for extended chars"
	} else {
		r.Status = "degraded"
		r.Notes = "box drawing characters may not render correctly"
		r.Workaround = "use ASCII characters (+, -, |) for borders"
	}
	return r
}

func ttCheckBraille(r CompatResult, p TerminalProfile) CompatResult {
	if p.UnicodeFull {
		r.Status = "full"
		r.Notes = "braille pattern characters fully supported"
	} else if p.BoxDrawing {
		r.Status = "partial"
		r.Notes = "basic braille patterns may work"
		r.Workaround = "use block characters as braille fallback"
	} else {
		r.Status = "unsupported"
		r.Notes = "braille characters not reliably rendered"
		r.Workaround = "use ASCII art for graphs"
	}
	return r
}

func ttCheckSparkline(r CompatResult, p TerminalProfile) CompatResult {
	if p.UnicodeFull {
		r.Status = "full"
		r.Notes = "sparkline block elements fully supported"
	} else if p.BoxDrawing {
		r.Status = "partial"
		r.Notes = "basic sparkline characters work"
		r.Workaround = "use simplified bar characters"
	} else {
		r.Status = "degraded"
		r.Notes = "sparkline characters may not render"
		r.Workaround = "use ASCII bar chart representation"
	}
	return r
}

func ttCheckMouse(r CompatResult, p TerminalProfile) CompatResult {
	if p.MouseSupport {
		r.Status = "full"
		r.Notes = "SGR mouse encoding supported"
	} else {
		r.Status = "unsupported"
		r.Notes = "no mouse event support"
		r.Workaround = "use keyboard navigation only"
	}
	return r
}

func ttCheckClipboard(r CompatResult, p TerminalProfile) CompatResult {
	if p.OSC52Clipboard {
		r.Status = "full"
		r.Notes = "OSC 52 clipboard integration"
	} else {
		r.Status = "unsupported"
		r.Notes = "no clipboard protocol support"
		r.Workaround = "rely on terminal's native copy/paste"
	}
	return r
}

func ttCheckResize(r CompatResult, p TerminalProfile) CompatResult {
	if p.ResizeEvents {
		r.Status = "full"
		r.Notes = "SIGWINCH emitted reliably on resize"
	} else {
		r.Status = "degraded"
		r.Notes = "resize events may be delayed or missed"
		r.Workaround = "poll terminal size periodically"
	}
	return r
}

func ttCheckPixelSizing(r CompatResult, p TerminalProfile) CompatResult {
	if p.PixelSizeQuery {
		r.Status = "full"
		r.Notes = "CSI 16t pixel size query supported"
	} else {
		r.Status = "unsupported"
		r.Notes = "pixel dimensions not queryable"
		r.Workaround = "estimate pixel size from cell count and common font metrics"
	}
	return r
}
