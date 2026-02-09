// Package shelltest validates generated shell integration scripts for
// prompt-pulse v2. It performs structural analysis -- pattern matching,
// syntax checking, version compatibility, and security auditing -- without
// executing any shell code.
//
// All private helpers are prefixed with "st" to avoid naming conflicts.
package shelltest

import (
	"regexp"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/shell"
)

// Pattern defines a required or forbidden string pattern in a shell script.
type Pattern struct {
	Name        string
	Regex       string // regex pattern string
	Required    bool   // true = must be present, false = must be absent
	Description string

	compiled *regexp.Regexp
}

// stCompile lazily compiles and returns the pattern's regexp.
func (p *Pattern) stCompile() *regexp.Regexp {
	if p.compiled == nil {
		p.compiled = regexp.MustCompile(p.Regex)
	}
	return p.compiled
}

// Matches reports whether the pattern matches anywhere in script.
func (p *Pattern) Matches(script string) bool {
	return p.stCompile().MatchString(script)
}

// PatternsFor returns the required/forbidden patterns for a shell type.
func PatternsFor(shellType shell.ShellType) []Pattern {
	switch shellType {
	case shell.Bash:
		return stBashPatterns()
	case shell.Zsh:
		return stZshPatterns()
	case shell.Fish:
		return stFishPatterns()
	case shell.Ksh:
		return stKshPatterns()
	default:
		return nil
	}
}

func stBashPatterns() []Pattern {
	return []Pattern{
		{
			Name:        "prompt_command",
			Regex:       `PROMPT_COMMAND`,
			Required:    true,
			Description: "Bash integration must use PROMPT_COMMAND for precmd hook",
		},
		{
			Name:        "bind",
			Regex:       `bind\b`,
			Required:    true,
			Description: "Bash integration must use bind for keybinding",
		},
		{
			Name:        "complete_c",
			Regex:       `complete -C`,
			Required:    true,
			Description: "Bash integration must use complete -C for completions",
		},
		{
			Name:        "pp_start",
			Regex:       `pp-start\(\)`,
			Required:    true,
			Description: "Bash integration must define pp-start() function",
		},
		{
			Name:        "pp_stop",
			Regex:       `pp-stop\(\)`,
			Required:    true,
			Description: "Bash integration must define pp-stop() function",
		},
		{
			Name:        "pp_status",
			Regex:       `pp-status\(\)`,
			Required:    true,
			Description: "Bash integration must define pp-status() function",
		},
		{
			Name:        "pp_banner",
			Regex:       `pp-banner\(\)`,
			Required:    true,
			Description: "Bash integration must define pp-banner() function",
		},
		{
			Name:        "quoted_binary",
			Regex:       `'[^']*'`,
			Required:    true,
			Description: "Binary path must be properly quoted",
		},
		{
			Name:        "bare_eval_injection",
			Regex:       `eval\s+\$\(`,
			Required:    false,
			Description: "eval $( without quoting is an injection risk",
		},
		{
			Name:        "unbalanced_braces",
			Regex:       `(?s)^[^{}]*\}`,
			Required:    false,
			Description: "Script must not start with unbalanced closing brace",
		},
	}
}

func stZshPatterns() []Pattern {
	return []Pattern{
		{
			Name:        "add_zsh_hook",
			Regex:       `add-zsh-hook`,
			Required:    true,
			Description: "Zsh integration must use add-zsh-hook for precmd",
		},
		{
			Name:        "zle_n",
			Regex:       `zle -N`,
			Required:    true,
			Description: "Zsh integration must register ZLE widget via zle -N",
		},
		{
			Name:        "bindkey",
			Regex:       `bindkey`,
			Required:    true,
			Description: "Zsh integration must use bindkey for keybinding",
		},
		{
			Name:        "dev_tty",
			Regex:       `/dev/tty`,
			Required:    true,
			Description: "Zsh TUI widget must redirect via /dev/tty",
		},
		{
			Name:        "pp_start",
			Regex:       `pp-start\(\)`,
			Required:    true,
			Description: "Zsh integration must define pp-start() function",
		},
		{
			Name:        "pp_stop",
			Regex:       `pp-stop\(\)`,
			Required:    true,
			Description: "Zsh integration must define pp-stop() function",
		},
		{
			Name:        "pp_status",
			Regex:       `pp-status\(\)`,
			Required:    true,
			Description: "Zsh integration must define pp-status() function",
		},
		{
			Name:        "bare_eval",
			Regex:       `eval\s+[^"]`,
			Required:    false,
			Description: "eval without quoted argument is a risk in zsh",
		},
	}
}

func stFishPatterns() []Pattern {
	return []Pattern{
		{
			Name:        "function_keyword",
			Regex:       `\bfunction\b`,
			Required:    true,
			Description: "Fish integration must use function keyword",
		},
		{
			Name:        "bind",
			Regex:       `\bbind\b`,
			Required:    true,
			Description: "Fish integration must use bind for keybinding",
		},
		{
			Name:        "complete_c",
			Regex:       `complete -c`,
			Required:    true,
			Description: "Fish integration must use complete -c for completions",
		},
		{
			Name:        "on_event_fish_prompt",
			Regex:       `--on-event fish_prompt`,
			Required:    true,
			Description: "Fish banner must use --on-event fish_prompt",
		},
		{
			Name:        "bind_m_default",
			Regex:       `bind\s+[^-]`,
			Required:    true,
			Description: "Fish integration must bind in default mode (implicit, no -M flag)",
		},
		{
			Name:        "bind_m_insert",
			Regex:       `-M insert`,
			Required:    true,
			Description: "Fish integration must bind in insert mode",
		},
		{
			Name:        "bind_m_visual",
			Regex:       `-M visual`,
			Required:    true,
			Description: "Fish integration must bind in visual mode",
		},
		{
			Name:        "pp_start",
			Regex:       `\bfunction pp-start\b`,
			Required:    true,
			Description: "Fish integration must define pp-start as fish function",
		},
		{
			Name:        "pp_stop",
			Regex:       `\bfunction pp-stop\b`,
			Required:    true,
			Description: "Fish integration must define pp-stop as fish function",
		},
		{
			Name:        "pp_status",
			Regex:       `\bfunction pp-status\b`,
			Required:    true,
			Description: "Fish integration must define pp-status as fish function",
		},
	}
}

func stKshPatterns() []Pattern {
	return []Pattern{
		{
			Name:        "ps1",
			Regex:       `PS1=`,
			Required:    true,
			Description: "Ksh integration must set PS1 for prompt",
		},
		{
			Name:        "trap",
			Regex:       `\btrap\b`,
			Required:    true,
			Description: "Ksh integration must use trap for keybinding",
		},
		{
			Name:        "debug_or_keybd",
			Regex:       `KEYBD`,
			Required:    true,
			Description: "Ksh integration must use KEYBD trap",
		},
		{
			Name:        "pp_start",
			Regex:       `pp-start\(\)`,
			Required:    true,
			Description: "Ksh integration must define pp-start() function",
		},
		{
			Name:        "pp_stop",
			Regex:       `pp-stop\(\)`,
			Required:    true,
			Description: "Ksh integration must define pp-stop() function",
		},
		{
			Name:        "pp_status",
			Regex:       `pp-status\(\)`,
			Required:    true,
			Description: "Ksh integration must define pp-status() function",
		},
	}
}
