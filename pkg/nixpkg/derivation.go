package nixpkg

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const derivationTemplate = `{ lib, buildGoModule, ... }:

buildGoModule rec {
  pname = "{{ .Name }}";
  version = "{{ .Version }}";

  src = ./.;

  vendorHash = "{{ .VendorHash }}";

  subPackages = [ "{{ .SubPackage }}" ];

  ldflags = [
    "-s" "-w"
    "-X main.version=${version}"
  ];

  meta = with lib; {
    description = "{{ .Description }}";
    homepage = "{{ .Homepage }}";
    license = licenses.{{ .LicenseNix }};
    platforms = [ {{ .PlatformsNix }} ];
    mainProgram = "{{ .Name }}";
  };
}
`

// derivationData is the template context for Nix derivation generation.
type derivationData struct {
	Name         string
	Version      string
	VendorHash   string
	SubPackage   string
	Description  string
	Homepage     string
	LicenseNix   string
	PlatformsNix string
}

// GenerateDerivation produces a complete buildGoModule Nix expression from
// the given PackageMeta and VendorInfo. The returned string is a valid
// Nix file suitable for writing to package.nix.
func GenerateDerivation(meta *PackageMeta, vendorInfo *VendorInfo) (string, error) {
	if meta == nil {
		return "", fmt.Errorf("meta is nil")
	}

	errs := ValidateMeta(meta)
	if len(errs) > 0 {
		return "", fmt.Errorf("invalid meta: %s", strings.Join(errs, "; "))
	}

	vendorHash := ""
	if vendorInfo != nil && vendorInfo.Hash != "" {
		vendorHash = npFormatNixHash(vendorInfo.Hash)
	}

	// Derive subPackage from MainPackage: "./cmd/prompt-pulse" -> "cmd/prompt-pulse"
	subPkg := strings.TrimPrefix(meta.MainPackage, "./")
	if subPkg == "" {
		subPkg = "."
	}

	// Format platform list as Nix string literals.
	var platforms []string
	for _, arch := range meta.Architectures {
		platforms = append(platforms, fmt.Sprintf("%q", arch))
	}

	data := derivationData{
		Name:         meta.Name,
		Version:      meta.Version,
		VendorHash:   vendorHash,
		SubPackage:   subPkg,
		Description:  meta.Description,
		Homepage:     meta.Homepage,
		LicenseNix:   npLicenseToNix(meta.License),
		PlatformsNix: strings.Join(platforms, " "),
	}

	tmpl, err := template.New("derivation").Parse(derivationTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// npBuildTemplate returns the raw derivation template string.
func npBuildTemplate() string {
	return derivationTemplate
}

// npLicenseToNix maps an SPDX license identifier to its nixpkgs name.
func npLicenseToNix(spdx string) string {
	m := map[string]string{
		"MIT":        "mit",
		"Apache-2.0": "asl20",
		"GPL-3.0":    "gpl3Only",
		"GPL-2.0":    "gpl2Only",
		"BSD-3":      "bsd3",
		"BSD-2":      "bsd2",
		"MPL-2.0":    "mpl20",
		"ISC":        "isc",
		"Unlicense":  "unlicense",
	}
	if nix, ok := m[spdx]; ok {
		return nix
	}
	return strings.ToLower(spdx)
}
