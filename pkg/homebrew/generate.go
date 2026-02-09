package homebrew

import (
	"bytes"
	"strings"
	"text/template"
	"unicode"
)

// GenerateFormula renders a complete Homebrew Ruby formula from the given config.
func GenerateFormula(config *FormulaConfig) (string, error) {
	tmpl, err := template.New("formula").Funcs(template.FuncMap{
		"pascalCase":  hbPascalCase,
		"depLine":     hbDepLine,
		"joinLdFlags": hbJoinLdFlags,
	}).Parse(hbBuildTemplate())
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// hbBuildTemplate returns the Ruby template string for a Homebrew formula.
func hbBuildTemplate() string {
	return `class {{ pascalCase .Name }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  license "{{ .License }}"
  head "https://gitlab.com/{{ .GitLabProject }}.git", branch: "{{ .HeadBranch }}"

{{ range .Dependencies }}  {{ depLine . }}
{{ end }}
  def install
    ldflags = %W[
{{ joinLdFlags .LdFlags }}    ]
    system "go", "build", *std_go_args(ldflags:), "./cmd/{{ .Name }}"
{{ if .ShellCompletions }}
    generate_completions_from_executable(bin/"{{ .Name }}", "completion")
{{ end }}  end
{{ if .DaemonService }}
  service do
    run [opt_bin/"{{ .Name }}", "daemon"]
    keep_alive true
    log_path var/"log/{{ .Name }}.log"
    error_log_path var/"log/{{ .Name }}-error.log"
  end
{{ end }}
  test do
    assert_match version.to_s, shell_output("#{bin}/{{ .Name }} --version")
    assert_match "{{ .Name }}", shell_output("#{bin}/{{ .Name }} banner --help")
  end
end
`
}

// hbPascalCase converts a kebab-case or snake_case name to PascalCase.
// For example "prompt-pulse" becomes "PromptPulse".
func hbPascalCase(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		if r == '-' || r == '_' {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// hbDepLine formats a FormulaDep as a Ruby depends_on line.
func hbDepLine(dep FormulaDep) string {
	switch dep.Type {
	case "build":
		return `depends_on "` + dep.Name + `" => :build`
	case "test":
		return `depends_on "` + dep.Name + `" => :test`
	case "optional":
		return `depends_on "` + dep.Name + `" => :optional`
	default:
		// runtime or unspecified
		return `depends_on "` + dep.Name + `"`
	}
}

// hbJoinLdFlags formats ldflags for the Ruby %W[] array, each indented with 6 spaces.
func hbJoinLdFlags(flags []string) string {
	var b strings.Builder
	for _, f := range flags {
		b.WriteString("      ")
		b.WriteString(f)
		b.WriteString("\n")
	}
	return b.String()
}
