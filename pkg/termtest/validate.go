package termtest

import "fmt"

// ValidateBoxDrawing checks if box drawing characters are safe to use
// with the given terminal profile.
func ValidateBoxDrawing(profile TerminalProfile) error {
	if !profile.BoxDrawing {
		return fmt.Errorf("terminal %q does not support box drawing characters", profile.Name)
	}
	return nil
}

// ValidateImageProtocol returns the best image protocol for the given
// terminal profile, or an error if no protocol is available.
func ValidateImageProtocol(profile TerminalProfile) (string, error) {
	proto := ttBestImageProtocol(profile)
	if proto == "none" {
		return proto, fmt.Errorf("terminal %q has no image rendering protocol", profile.Name)
	}
	return proto, nil
}

// ValidateColorDepth returns the effective color depth for the given
// terminal profile. Returns 24 for truecolor terminals, 256 otherwise.
func ValidateColorDepth(profile TerminalProfile) int {
	if profile.ColorDepth == 24 {
		return 24
	}
	return 256
}

// ttBestImageProtocol selects the optimal image protocol for a terminal.
// Priority: kitty > iterm2 > sixel > halfblock > none.
func ttBestImageProtocol(profile TerminalProfile) string {
	switch profile.ImageProtocol {
	case "kitty":
		return "kitty"
	case "iterm2":
		return "iterm2"
	case "sixel":
		return "sixel"
	case "halfblock":
		return "halfblock"
	default:
		return "none"
	}
}

// ttFallbackStrategy returns the degradation strategy for a feature that
// is not fully supported on the given terminal profile.
func ttFallbackStrategy(feature string, profile TerminalProfile) string {
	results := CheckCompat(profile)
	for _, r := range results {
		if r.Feature == feature {
			if r.Workaround != "" {
				return r.Workaround
			}
			if r.Status == "full" {
				return "no fallback needed"
			}
			return "no known workaround"
		}
	}
	return "unknown feature"
}
