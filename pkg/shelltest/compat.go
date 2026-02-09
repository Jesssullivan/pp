package shelltest

import (
	"fmt"
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/shell"
)

// ShellVersion describes a shell version for compatibility testing.
type ShellVersion struct {
	Shell    shell.ShellType
	Version  string   // e.g., "5.2", "4.4", "3.6"
	Features []string // features supported at this version
}

// KnownVersions returns the shell versions we test against.
func KnownVersions() []ShellVersion {
	return []ShellVersion{
		// Bash versions
		{Shell: shell.Bash, Version: "3.2", Features: []string{"PROMPT_COMMAND", "complete", "bind"}},
		{Shell: shell.Bash, Version: "4.3", Features: []string{"PROMPT_COMMAND", "complete", "bind", "bind-x"}},
		{Shell: shell.Bash, Version: "4.4", Features: []string{"PROMPT_COMMAND", "complete", "bind", "bind-x", "PS0"}},
		{Shell: shell.Bash, Version: "5.0", Features: []string{"PROMPT_COMMAND", "complete", "bind", "bind-x", "PS0", "EPOCHREALTIME"}},
		{Shell: shell.Bash, Version: "5.2", Features: []string{"PROMPT_COMMAND", "complete", "bind", "bind-x", "PS0", "EPOCHREALTIME"}},

		// Zsh versions
		{Shell: shell.Zsh, Version: "5.0", Features: []string{"zle", "bindkey", "compdef"}},
		{Shell: shell.Zsh, Version: "5.1", Features: []string{"zle", "bindkey", "compdef", "add-zsh-hook"}},
		{Shell: shell.Zsh, Version: "5.8", Features: []string{"zle", "bindkey", "compdef", "add-zsh-hook"}},
		{Shell: shell.Zsh, Version: "5.9", Features: []string{"zle", "bindkey", "compdef", "add-zsh-hook"}},

		// Fish versions
		{Shell: shell.Fish, Version: "3.0", Features: []string{"function", "bind", "complete", "string"}},
		{Shell: shell.Fish, Version: "3.3", Features: []string{"function", "bind", "complete", "string", "bind-mode-insert"}},
		{Shell: shell.Fish, Version: "3.6", Features: []string{"function", "bind", "complete", "string", "bind-mode-insert", "path"}},

		// Ksh versions
		{Shell: shell.Ksh, Version: "88", Features: []string{"PS1", "trap-DEBUG", "typeset"}},
		{Shell: shell.Ksh, Version: "93", Features: []string{"PS1", "trap-DEBUG", "typeset", "KEYBD", "nameref", ".sh.edchar"}},
	}
}

// CheckVersionCompat checks if a generated script uses features not
// available in the specified shell version. Returns a list of warnings
// for features used but not available at that version.
func CheckVersionCompat(shellType shell.ShellType, version string, script string) []string {
	switch shellType {
	case shell.Bash:
		return stCheckBashVersionCompat(version, script)
	case shell.Zsh:
		return stCheckZshVersionCompat(version, script)
	case shell.Fish:
		return stCheckFishVersionCompat(version, script)
	case shell.Ksh:
		return stCheckKshVersionCompat(version, script)
	default:
		return []string{fmt.Sprintf("unknown shell type: %s", shellType)}
	}
}

func stCheckBashVersionCompat(version string, script string) []string {
	var warnings []string

	// Bash 4.3 and earlier: no PS0 support.
	if stVersionLess(version, "4.4") {
		if strings.Contains(script, "PS0=") || strings.Contains(script, "PS0 =") {
			warnings = append(warnings, fmt.Sprintf(
				"PS0 is not available in bash %s (added in 4.4)", version))
		}
	}

	// Bash 3.x: no bind -x.
	if stVersionLess(version, "4.0") {
		if strings.Contains(script, "bind -x") {
			warnings = append(warnings, fmt.Sprintf(
				"bind -x is not available in bash %s (added in 4.0)", version))
		}
	}

	// Bash < 5.0: no EPOCHREALTIME.
	if stVersionLess(version, "5.0") {
		if strings.Contains(script, "EPOCHREALTIME") {
			warnings = append(warnings, fmt.Sprintf(
				"EPOCHREALTIME is not available in bash %s (added in 5.0)", version))
		}
	}

	return warnings
}

func stCheckZshVersionCompat(version string, script string) []string {
	var warnings []string

	// Zsh 5.0: add-zsh-hook not built-in (requires autoload).
	if stVersionLess(version, "5.1") {
		if strings.Contains(script, "add-zsh-hook") && !strings.Contains(script, "autoload") {
			warnings = append(warnings, fmt.Sprintf(
				"add-zsh-hook requires explicit autoload in zsh %s", version))
		}
	}

	return warnings
}

func stCheckFishVersionCompat(version string, script string) []string {
	var warnings []string

	// Fish 3.0: different bind syntax for modes.
	if stVersionLess(version, "3.1") {
		if strings.Contains(script, "-M insert") || strings.Contains(script, "-M visual") {
			warnings = append(warnings, fmt.Sprintf(
				"bind -M <mode> syntax may differ in fish %s", version))
		}
	}

	return warnings
}

func stCheckKshVersionCompat(version string, script string) []string {
	var warnings []string

	// Ksh88: no KEYBD trap.
	if version == "88" {
		if strings.Contains(script, "KEYBD") {
			warnings = append(warnings, "KEYBD trap is not available in ksh88 (ksh93 only)")
		}
		if strings.Contains(script, ".sh.edchar") {
			warnings = append(warnings, ".sh.edchar is not available in ksh88 (ksh93 only)")
		}
	}

	return warnings
}

// stVersionLess reports whether version a is less than version b using
// simple dot-separated numeric comparison.
func stVersionLess(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(aParts) {
			av = stParseVersionPart(aParts[i])
		}
		if i < len(bParts) {
			bv = stParseVersionPart(bParts[i])
		}
		if av < bv {
			return true
		}
		if av > bv {
			return false
		}
	}
	return false
}

// stParseVersionPart converts a version segment string to int.
func stParseVersionPart(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}
