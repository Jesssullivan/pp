package nixpkg

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

// DevShellConfig defines the packages, hooks, and environment variables
// for a Nix development shell.
type DevShellConfig struct {
	// Packages lists Nix package attribute names to include in the shell.
	Packages []string

	// ShellHook is a bash script executed when entering the shell.
	ShellHook string

	// EnvVars maps environment variable names to their values.
	EnvVars map[string]string
}

const devShellTemplate = `{ pkgs, ... }:

pkgs.mkShell {
  buildInputs = with pkgs; [
{{ range .Packages }}    {{ . }}
{{ end }}  ];
{{ if .EnvLines }}
{{ range .EnvLines }}  {{ . }}
{{ end }}{{ end }}{{ if .ShellHook }}
  shellHook = ''
    {{ .ShellHook }}
  '';
{{ end }}}
`

// devShellData is the template context for dev shell generation.
type devShellData struct {
	Packages  []string
	ShellHook string
	EnvLines  []string
}

// DefaultDevShell returns a DevShellConfig with standard prompt-pulse v2
// development dependencies, compiler flags, and tool configuration.
func DefaultDevShell() *DevShellConfig {
	return &DevShellConfig{
		Packages: []string{
			"go_1_23",
			"gopls",
			"golangci-lint",
			"delve",
		},
		ShellHook: `export GOFLAGS="-mod=mod"
    export CGO_ENABLED=0`,
		EnvVars: map[string]string{
			"CGO_ENABLED": "0",
			"GOFLAGS":     "-mod=mod",
		},
	}
}

// GenerateDevShell produces a Nix devShell expression from the given config.
// The output is suitable for use in a flake's devShells output or as a
// standalone shell.nix file.
func GenerateDevShell(config *DevShellConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config is nil")
	}
	if len(config.Packages) == 0 {
		return "", fmt.Errorf("at least one package is required")
	}

	// Build sorted env var lines for deterministic output.
	var envLines []string
	if len(config.EnvVars) > 0 {
		keys := make([]string, 0, len(config.EnvVars))
		for k := range config.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			envLines = append(envLines, fmt.Sprintf("%s = %q;", k, config.EnvVars[k]))
		}
	}

	data := devShellData{
		Packages:  config.Packages,
		ShellHook: strings.TrimSpace(config.ShellHook),
		EnvLines:  envLines,
	}

	tmpl, err := template.New("devshell").Parse(devShellTemplate)
	if err != nil {
		return "", fmt.Errorf("parse devshell template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute devshell template: %w", err)
	}

	return buf.String(), nil
}
