package homebrew

import (
	"bytes"
	"text/template"
)

// BottleConfig holds configuration for Homebrew bottle (prebuilt binary) blocks.
type BottleConfig struct {
	// RootURL is the base URL where bottles are hosted.
	RootURL string

	// Architectures lists per-platform bottle entries.
	Architectures []BottleArch

	// Rebuild is the bottle rebuild counter (usually 0).
	Rebuild int
}

// BottleArch describes a single architecture entry in a bottle block.
type BottleArch struct {
	// OS is the operating system (e.g. "macos", "linux").
	OS string

	// Arch is the CPU architecture (e.g. "arm64", "x86_64").
	Arch string

	// SHA256 is the sha256 checksum of the bottle tarball.
	SHA256 string

	// Cellar is the cellar path (e.g. ":any_skip_relocation").
	Cellar string
}

// DefaultBottles returns a BottleConfig with default entries for arm64_sonoma
// and x86_64_linux, using placeholder SHA256 values.
func DefaultBottles() *BottleConfig {
	return &BottleConfig{
		RootURL: "https://gitlab.com/api/v4/projects/tinyland%2Flab%2Fprompt-pulse/packages/generic/bottles",
		Architectures: []BottleArch{
			{
				OS:     "macos",
				Arch:   "arm64",
				SHA256: "PLACEHOLDER_SHA256_ARM64_SONOMA",
				Cellar: ":any_skip_relocation",
			},
			{
				OS:     "linux",
				Arch:   "x86_64",
				SHA256: "PLACEHOLDER_SHA256_X86_64_LINUX",
				Cellar: ":any_skip_relocation",
			},
		},
		Rebuild: 0,
	}
}

// GenerateBottleBlock renders the bottle do...end Ruby block from the config.
func GenerateBottleBlock(config *BottleConfig) (string, error) {
	tmpl, err := template.New("bottle").Funcs(template.FuncMap{
		"bottleTag": func(a BottleArch) string { return hbBottleTag(a.OS, a.Arch) },
	}).Parse(hbBottleTemplate())
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// hbBottleTag maps an OS and architecture pair to a Homebrew bottle tag.
// For example, ("macos", "arm64") returns "arm64_sonoma".
func hbBottleTag(os, arch string) string {
	key := os + "/" + arch
	tags := map[string]string{
		"macos/arm64":  "arm64_sonoma",
		"macos/x86_64": "sonoma",
		"linux/x86_64": "x86_64_linux",
		"linux/arm64":  "linux",
	}
	if tag, ok := tags[key]; ok {
		return tag
	}
	return arch + "_" + os
}

// hbBottleTemplate returns the Ruby template for a bottle block.
func hbBottleTemplate() string {
	return `  bottle do
    root_url "{{ .RootURL }}"
{{ if gt .Rebuild 0 }}    rebuild {{ .Rebuild }}
{{ end }}{{ range .Architectures }}    sha256 cellar: {{ .Cellar }}, {{ bottleTag . }}: "{{ .SHA256 }}"
{{ end }}  end
`
}
