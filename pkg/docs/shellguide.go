package docs

import (
	"fmt"
	"strings"
)

// ShellGuide holds shell integration documentation for all supported shells.
type ShellGuide struct {
	// Shells lists integration details for each supported shell.
	Shells []ShellInfo
}

// ShellInfo documents the integration for a single shell.
type ShellInfo struct {
	// Name is the shell name (e.g., "Bash").
	Name string

	// ConfigFile is the path to the shell's config file.
	ConfigFile string

	// SetupCommand is the shell snippet to add to the config file.
	SetupCommand string

	// HookType describes how the shell hook is installed.
	HookType string

	// Features lists capabilities of this shell integration.
	Features []string

	// Caveats lists known limitations or gotchas.
	Caveats []string

	// Example is a complete config file snippet.
	Example string
}

// dcGenerateShellGuide builds the shell integration guide from embedded knowledge.
func dcGenerateShellGuide() *ShellGuide {
	return &ShellGuide{
		Shells: []ShellInfo{
			dcBashShell(),
			dcZshShell(),
			dcFishShell(),
			dcKshShell(),
		},
	}
}

// dcRenderShellMarkdown renders a ShellGuide as a Markdown document.
func dcRenderShellMarkdown(guide *ShellGuide) string {
	var b strings.Builder

	b.WriteString("# Shell Integration Guide\n\n")
	b.WriteString("prompt-pulse integrates with 4 shells: Bash, Zsh, Fish, and Ksh.\n\n")
	b.WriteString("Run `prompt-pulse shell init <shell>` to generate the setup snippet.\n\n")

	for _, s := range guide.Shells {
		b.WriteString(fmt.Sprintf("## %s\n\n", s.Name))
		b.WriteString(fmt.Sprintf("**Config file:** `%s`\n\n", s.ConfigFile))
		b.WriteString(fmt.Sprintf("**Hook type:** %s\n\n", s.HookType))

		b.WriteString("### Setup\n\n")
		b.WriteString("Add to your shell config:\n\n")
		b.WriteString("```sh\n")
		b.WriteString(s.SetupCommand)
		b.WriteString("\n```\n\n")

		b.WriteString("### Features\n\n")
		for _, f := range s.Features {
			b.WriteString(fmt.Sprintf("- %s\n", f))
		}
		b.WriteString("\n")

		if len(s.Caveats) > 0 {
			b.WriteString("### Caveats\n\n")
			for _, c := range s.Caveats {
				b.WriteString(fmt.Sprintf("- %s\n", c))
			}
			b.WriteString("\n")
		}

		b.WriteString("### Full Example\n\n")
		b.WriteString("```sh\n")
		b.WriteString(s.Example)
		b.WriteString("\n```\n\n")
	}

	return b.String()
}

func dcBashShell() ShellInfo {
	return ShellInfo{
		Name:       "Bash",
		ConfigFile: "~/.bashrc",
		SetupCommand: `eval "$(prompt-pulse shell init bash)"`,
		HookType:   "PROMPT_COMMAND append",
		Features: []string{
			"Banner on shell startup via PROMPT_COMMAND",
			"TUI toggle hotkey via `bind -x`",
			"Tab completions via `complete -C`",
			"Daemon auto-start on first prompt",
		},
		Caveats: []string{
			"PROMPT_COMMAND is appended, not replaced, to preserve existing hooks",
			"Requires Bash 4.0+ for `bind -x` support",
			"Completions require bash-completion package on some systems",
		},
		Example: `# prompt-pulse shell integration for Bash
# Add to ~/.bashrc

# Initialize prompt-pulse hooks
eval "$(prompt-pulse shell init bash)"

# The init command sets up:
# 1. PROMPT_COMMAND hook for banner display
# 2. Ctrl+P keybinding for TUI toggle via bind -x
# 3. Tab completion via complete -C
# 4. Daemon auto-start check`,
	}
}

func dcZshShell() ShellInfo {
	return ShellInfo{
		Name:       "Zsh",
		ConfigFile: "~/.zshrc",
		SetupCommand: `eval "$(prompt-pulse shell init zsh)"`,
		HookType:   "add-zsh-hook precmd",
		Features: []string{
			"Banner on shell startup via precmd hook",
			"TUI toggle hotkey via ZLE widget with /dev/tty",
			"Tab completions via compdef",
			"Daemon auto-start on first precmd",
		},
		Caveats: []string{
			"ZLE widget reads from /dev/tty to avoid interfering with line editing",
			"Must be loaded after compinit for completions to work",
			"Some themes may conflict with precmd hook ordering",
		},
		Example: `# prompt-pulse shell integration for Zsh
# Add to ~/.zshrc (after compinit)

# Initialize prompt-pulse hooks
eval "$(prompt-pulse shell init zsh)"

# The init command sets up:
# 1. precmd hook via add-zsh-hook for banner display
# 2. ZLE widget for Ctrl+P TUI toggle (reads /dev/tty)
# 3. Completion function via compdef
# 4. Daemon auto-start check`,
	}
}

func dcFishShell() ShellInfo {
	return ShellInfo{
		Name:       "Fish",
		ConfigFile: "~/.config/fish/config.fish",
		SetupCommand: `prompt-pulse shell init fish | source`,
		HookType:   "--on-event fish_prompt",
		Features: []string{
			"Banner on shell startup via fish_prompt event",
			"TUI toggle hotkey via bind in normal, insert, and visual modes",
			"Tab completions via complete command",
			"Daemon auto-start on first prompt event",
		},
		Caveats: []string{
			"Fish uses `source` instead of `eval` for initialization",
			"Keybindings must be registered for all three modes (normal, insert, visual)",
			"Fish completions use a different syntax than Bash/Zsh",
		},
		Example: `# prompt-pulse shell integration for Fish
# Add to ~/.config/fish/config.fish

# Initialize prompt-pulse hooks
prompt-pulse shell init fish | source

# The init command sets up:
# 1. fish_prompt event handler for banner display
# 2. Keybindings in normal, insert, and visual modes
# 3. Fish-native completions
# 4. Daemon auto-start check`,
	}
}

func dcKshShell() ShellInfo {
	return ShellInfo{
		Name:       "Ksh",
		ConfigFile: "~/.kshrc",
		SetupCommand: `eval "$(prompt-pulse shell init ksh)"`,
		HookType:   "PS1 command substitution",
		Features: []string{
			"Banner on shell startup via PS1 command substitution",
			"TUI toggle via trap KEYBD",
			"Daemon status via trap DEBUG",
			"Minimal footprint for POSIX compatibility",
		},
		Caveats: []string{
			"Ksh93 or mksh required; pdksh is not supported",
			"KEYBD trap is ksh93-specific and not available in all implementations",
			"No native tab completion support; uses basic word expansion",
		},
		Example: `# prompt-pulse shell integration for Ksh
# Add to ~/.kshrc

# Initialize prompt-pulse hooks
eval "$(prompt-pulse shell init ksh)"

# The init command sets up:
# 1. PS1 with embedded command substitution for banner
# 2. KEYBD trap for TUI toggle
# 3. DEBUG trap for daemon status check
# 4. Daemon auto-start on first prompt`,
	}
}
