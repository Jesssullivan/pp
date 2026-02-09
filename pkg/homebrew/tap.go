package homebrew

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

// TapConfig holds metadata about a Homebrew tap repository.
type TapConfig struct {
	// Organization is the GitHub/GitLab org or user (e.g. "tinyland").
	Organization string

	// TapName is the tap name (e.g. "homebrew-tap").
	TapName string

	// FormulaDir is the subdirectory within the tap repo for formula files.
	FormulaDir string
}

// DefaultTap returns a TapConfig with tinyland defaults.
func DefaultTap() *TapConfig {
	return &TapConfig{
		Organization: "tinyland",
		TapName:      "homebrew-tap",
		FormulaDir:   "Formula",
	}
}

// GenerateTapReadme renders a README.md for a Homebrew tap repository.
func GenerateTapReadme(tap *TapConfig, formulas []string) (string, error) {
	tmpl, err := template.New("readme").Parse(hbTapReadmeTemplate())
	if err != nil {
		return "", err
	}

	data := struct {
		*TapConfig
		Formulas []string
	}{tap, formulas}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// hbFormulaPath computes the filesystem path for a formula file within a tap repo.
func hbFormulaPath(tap *TapConfig, formulaName string) string {
	return filepath.Join(tap.FormulaDir, formulaName+".rb")
}

// ValidateTapStructure checks that a tap directory has the correct structure
// and returns a list of issues found. An empty slice means valid.
func ValidateTapStructure(tapDir string) []string {
	var errs []string

	info, err := os.Stat(tapDir)
	if err != nil {
		errs = append(errs, "tap directory does not exist: "+tapDir)
		return errs
	}
	if !info.IsDir() {
		errs = append(errs, "tap path is not a directory: "+tapDir)
		return errs
	}

	formulaDir := filepath.Join(tapDir, "Formula")
	if fi, err := os.Stat(formulaDir); err != nil || !fi.IsDir() {
		errs = append(errs, "missing Formula/ directory in tap")
	}

	readmePath := filepath.Join(tapDir, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		errs = append(errs, "missing README.md in tap")
	}

	return errs
}

// hbTapReadmeTemplate returns the markdown template for a tap README.
func hbTapReadmeTemplate() string {
	return `# {{ .Organization }}/{{ .TapName }}

Homebrew tap for {{ .Organization }} projects.

## Installation

` + "```bash" + `
brew tap {{ .Organization }}/{{ .TapName }}
` + "```" + `

## Available Formulas

{{ range .Formulas }}- ` + "`{{ . }}`" + `
{{ end }}
## Usage

` + "```bash" + `
brew install {{ .Organization }}/{{ .TapName }}/<formula>
` + "```" + `
`
}
