package nixpkg

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const overlayTemplate = `final: prev: {
  {{ .Name }} = final.callPackage ./package.nix { };
}
`

const flakeInputTemplate = `    {{ .Name }} = {
      url = "{{ .Homepage }}";
      flake = false;
    };
`

// overlayData is the template context for overlay generation.
type overlayData struct {
	Name     string
	Homepage string
}

// GenerateOverlay produces a Nix overlay expression that adds the package
// to nixpkgs via callPackage. The overlay expects a package.nix file in
// the same directory.
func GenerateOverlay(meta *PackageMeta) (string, error) {
	if meta == nil {
		return "", fmt.Errorf("meta is nil")
	}
	if meta.Name == "" {
		return "", fmt.Errorf("package name is required")
	}

	data := overlayData{
		Name:     meta.Name,
		Homepage: meta.Homepage,
	}

	tmpl, err := template.New("overlay").Parse(overlayTemplate)
	if err != nil {
		return "", fmt.Errorf("parse overlay template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute overlay template: %w", err)
	}

	return buf.String(), nil
}

// GenerateFlakeInput produces a Nix flake input block for the package.
// This can be added to a flake.nix inputs section.
func GenerateFlakeInput(meta *PackageMeta) (string, error) {
	if meta == nil {
		return "", fmt.Errorf("meta is nil")
	}
	if meta.Name == "" {
		return "", fmt.Errorf("package name is required for flake input")
	}

	errs := ValidateMeta(meta)
	// Only care about name and homepage for flake input; filter relevant errors.
	for _, e := range errs {
		if strings.Contains(e, "homepage") {
			return "", fmt.Errorf("invalid meta: %s", e)
		}
	}

	data := overlayData{
		Name:     meta.Name,
		Homepage: meta.Homepage,
	}

	tmpl, err := template.New("flakeInput").Parse(flakeInputTemplate)
	if err != nil {
		return "", fmt.Errorf("parse flake input template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute flake input template: %w", err)
	}

	return buf.String(), nil
}
