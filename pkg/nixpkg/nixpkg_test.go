package nixpkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Package metadata tests
// ---------------------------------------------------------------------------

func TestDefaultMeta(t *testing.T) {
	m := DefaultMeta()

	if m.Name != "prompt-pulse" {
		t.Errorf("Name = %q, want %q", m.Name, "prompt-pulse")
	}
	if m.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", m.Version, "2.0.0")
	}
	if m.License != "MIT" {
		t.Errorf("License = %q, want %q", m.License, "MIT")
	}
	if m.Homepage != "https://gitlab.com/tinyland/lab/prompt-pulse" {
		t.Errorf("Homepage = %q, unexpected", m.Homepage)
	}
	if m.MainPackage != "./cmd/prompt-pulse" {
		t.Errorf("MainPackage = %q, want %q", m.MainPackage, "./cmd/prompt-pulse")
	}
	if len(m.Architectures) != 3 {
		t.Errorf("Architectures count = %d, want 3", len(m.Architectures))
	}
	if len(m.Dependencies) == 0 {
		t.Error("Dependencies should not be empty")
	}
}

func TestValidateMeta_Valid(t *testing.T) {
	m := DefaultMeta()
	errs := ValidateMeta(m)
	if len(errs) != 0 {
		t.Errorf("ValidateMeta(DefaultMeta()) returned errors: %v", errs)
	}
}

func TestValidateMeta_MissingFields(t *testing.T) {
	m := &PackageMeta{}
	errs := ValidateMeta(m)

	// Should report errors for all required fields.
	expected := []string{"name", "version", "description", "license", "homepage", "main package", "architecture", "dependency"}
	for _, keyword := range expected {
		found := false
		for _, e := range errs {
			if strings.Contains(strings.ToLower(e), keyword) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error mentioning %q, got: %v", keyword, errs)
		}
	}
}

func TestValidateMeta_PartialMissing(t *testing.T) {
	m := DefaultMeta()
	m.Version = ""
	m.License = ""

	errs := ValidateMeta(m)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateMeta_CustomMeta(t *testing.T) {
	m := &PackageMeta{
		Name:          "my-tool",
		Version:       "3.1.0",
		Description:   "A custom tool",
		License:       "Apache-2.0",
		Homepage:      "https://example.com",
		MainPackage:   ".",
		Architectures: []string{"x86_64-linux"},
		Dependencies:  []string{"go_1_23"},
	}
	errs := ValidateMeta(m)
	if len(errs) != 0 {
		t.Errorf("ValidateMeta returned errors for valid custom meta: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Vendor info tests
// ---------------------------------------------------------------------------

func TestComputeVendorInfo(t *testing.T) {
	testdata := filepath.Join("testdata")
	info, err := npComputeVendorInfo(testdata)
	if err != nil {
		t.Fatalf("npComputeVendorInfo: %v", err)
	}

	if info.GoVersion != "1.23.4" {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, "1.23.4")
	}
	if info.NumDeps != 8 {
		t.Errorf("NumDeps = %d, want 8", info.NumDeps)
	}
	if info.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestComputeVendorInfo_MissingGoMod(t *testing.T) {
	_, err := npComputeVendorInfo("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
}

func TestFormatNixHash_Plain(t *testing.T) {
	h := npFormatNixHash("abc123def456")
	if h != "sha256-abc123def456" {
		t.Errorf("npFormatNixHash = %q, want %q", h, "sha256-abc123def456")
	}
}

func TestFormatNixHash_AlreadyPrefixed(t *testing.T) {
	h := npFormatNixHash("sha256-already")
	if h != "sha256-already" {
		t.Errorf("npFormatNixHash = %q, want %q", h, "sha256-already")
	}
}

func TestCountDeps_FromTestdata(t *testing.T) {
	count, err := npCountDeps(filepath.Join("testdata", "go.sum"))
	if err != nil {
		t.Fatalf("npCountDeps: %v", err)
	}
	if count != 8 {
		t.Errorf("npCountDeps = %d, want 8", count)
	}
}

func TestCountDeps_MissingFile(t *testing.T) {
	count, err := npCountDeps("/nonexistent/go.sum")
	if err != nil {
		t.Fatalf("npCountDeps should return 0 for missing file, got error: %v", err)
	}
	if count != 0 {
		t.Errorf("npCountDeps = %d, want 0 for missing file", count)
	}
}

func TestValidateVendorDir(t *testing.T) {
	// Create a temporary vendor directory with valid structure.
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor")
	os.MkdirAll(filepath.Join(vendorDir, "github.com"), 0o755)
	os.WriteFile(filepath.Join(vendorDir, "modules.txt"), []byte("# vendor\n"), 0o644)
	os.WriteFile(filepath.Join(vendorDir, "github.com", "dummy.go"), []byte("package dummy\n"), 0o644)

	err := npValidateVendorDir(vendorDir)
	if err != nil {
		t.Errorf("npValidateVendorDir: %v", err)
	}
}

func TestValidateVendorDir_MissingModulesTxt(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor")
	os.MkdirAll(vendorDir, 0o755)

	err := npValidateVendorDir(vendorDir)
	if err == nil {
		t.Error("expected error for missing modules.txt")
	}
}

func TestValidateVendorDir_EmptySubdir(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor")
	os.MkdirAll(filepath.Join(vendorDir, "emptymod"), 0o755)
	os.WriteFile(filepath.Join(vendorDir, "modules.txt"), []byte("# vendor\n"), 0o644)

	err := npValidateVendorDir(vendorDir)
	if err == nil {
		t.Error("expected error for empty subdirectory")
	}
	if !strings.Contains(err.Error(), "empty vendor subdirectory") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateVendorDir_NotExists(t *testing.T) {
	err := npValidateVendorDir("/nonexistent/vendor")
	if err == nil {
		t.Error("expected error for nonexistent vendor dir")
	}
}

// ---------------------------------------------------------------------------
// Derivation generation tests
// ---------------------------------------------------------------------------

func TestGenerateDerivation_Default(t *testing.T) {
	meta := DefaultMeta()
	vi := &VendorInfo{Hash: "abc123"}

	out, err := GenerateDerivation(meta, vi)
	if err != nil {
		t.Fatalf("GenerateDerivation: %v", err)
	}

	checks := []string{
		`pname = "prompt-pulse"`,
		`version = "2.0.0"`,
		`vendorHash = "sha256-abc123"`,
		`subPackages = [ "cmd/prompt-pulse" ]`,
		"-X main.version=${version}",
		"license = licenses.mit",
		`mainProgram = "prompt-pulse"`,
		"buildGoModule rec {",
		`"aarch64-darwin"`,
		`"x86_64-linux"`,
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("derivation missing %q\n---output---\n%s", c, out)
		}
	}
}

func TestGenerateDerivation_CustomMeta(t *testing.T) {
	meta := &PackageMeta{
		Name:          "my-app",
		Version:       "1.5.0",
		Description:   "My custom app",
		License:       "Apache-2.0",
		Homepage:      "https://example.com",
		MainPackage:   ".",
		Architectures: []string{"x86_64-linux"},
		Dependencies:  []string{"go_1_23"},
	}
	vi := &VendorInfo{Hash: "sha256-custom"}

	out, err := GenerateDerivation(meta, vi)
	if err != nil {
		t.Fatalf("GenerateDerivation: %v", err)
	}

	if !strings.Contains(out, `pname = "my-app"`) {
		t.Error("missing custom pname")
	}
	if !strings.Contains(out, `version = "1.5.0"`) {
		t.Error("missing custom version")
	}
	if !strings.Contains(out, "licenses.asl20") {
		t.Error("missing Apache license mapping")
	}
}

func TestGenerateDerivation_NilMeta(t *testing.T) {
	_, err := GenerateDerivation(nil, nil)
	if err == nil {
		t.Error("expected error for nil meta")
	}
}

func TestGenerateDerivation_InvalidMeta(t *testing.T) {
	_, err := GenerateDerivation(&PackageMeta{}, nil)
	if err == nil {
		t.Error("expected error for empty meta")
	}
}

func TestGenerateDerivation_NilVendorInfo(t *testing.T) {
	meta := DefaultMeta()
	out, err := GenerateDerivation(meta, nil)
	if err != nil {
		t.Fatalf("GenerateDerivation: %v", err)
	}
	// vendorHash should be empty string.
	if !strings.Contains(out, `vendorHash = ""`) {
		t.Error("expected empty vendorHash when vendorInfo is nil")
	}
}

func TestGenerateDerivation_BalancedBraces(t *testing.T) {
	meta := DefaultMeta()
	out, err := GenerateDerivation(meta, &VendorInfo{Hash: "test"})
	if err != nil {
		t.Fatalf("GenerateDerivation: %v", err)
	}

	opens := strings.Count(out, "{")
	closes := strings.Count(out, "}")
	if opens != closes {
		t.Errorf("unbalanced braces: %d opens, %d closes", opens, closes)
	}

	openBrackets := strings.Count(out, "[")
	closeBrackets := strings.Count(out, "]")
	if openBrackets != closeBrackets {
		t.Errorf("unbalanced brackets: %d opens, %d closes", openBrackets, closeBrackets)
	}
}

func TestBuildTemplate(t *testing.T) {
	tmpl := npBuildTemplate()
	if tmpl == "" {
		t.Error("npBuildTemplate returned empty string")
	}
	if !strings.Contains(tmpl, "buildGoModule") {
		t.Error("template missing buildGoModule")
	}
}

func TestLicenseToNix(t *testing.T) {
	tests := []struct {
		spdx string
		want string
	}{
		{"MIT", "mit"},
		{"Apache-2.0", "asl20"},
		{"GPL-3.0", "gpl3Only"},
		{"BSD-3", "bsd3"},
		{"ISC", "isc"},
		{"Unknown-License", "unknown-license"},
	}
	for _, tt := range tests {
		got := npLicenseToNix(tt.spdx)
		if got != tt.want {
			t.Errorf("npLicenseToNix(%q) = %q, want %q", tt.spdx, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Overlay generation tests
// ---------------------------------------------------------------------------

func TestGenerateOverlay(t *testing.T) {
	meta := DefaultMeta()
	out, err := GenerateOverlay(meta)
	if err != nil {
		t.Fatalf("GenerateOverlay: %v", err)
	}

	if !strings.Contains(out, "final: prev:") {
		t.Error("missing overlay function signature")
	}
	if !strings.Contains(out, "prompt-pulse = final.callPackage ./package.nix") {
		t.Error("missing callPackage invocation")
	}
}

func TestGenerateOverlay_NilMeta(t *testing.T) {
	_, err := GenerateOverlay(nil)
	if err == nil {
		t.Error("expected error for nil meta")
	}
}

func TestGenerateOverlay_EmptyName(t *testing.T) {
	_, err := GenerateOverlay(&PackageMeta{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestGenerateFlakeInput(t *testing.T) {
	meta := DefaultMeta()
	out, err := GenerateFlakeInput(meta)
	if err != nil {
		t.Fatalf("GenerateFlakeInput: %v", err)
	}

	if !strings.Contains(out, "prompt-pulse =") {
		t.Error("missing flake input name")
	}
	if !strings.Contains(out, "flake = false") {
		t.Error("missing flake = false")
	}
	if !strings.Contains(out, meta.Homepage) {
		t.Error("missing homepage URL")
	}
}

func TestGenerateFlakeInput_NilMeta(t *testing.T) {
	_, err := GenerateFlakeInput(nil)
	if err == nil {
		t.Error("expected error for nil meta")
	}
}

// ---------------------------------------------------------------------------
// DevShell tests
// ---------------------------------------------------------------------------

func TestDefaultDevShell(t *testing.T) {
	ds := DefaultDevShell()

	if len(ds.Packages) != 4 {
		t.Errorf("Packages count = %d, want 4", len(ds.Packages))
	}
	if ds.Packages[0] != "go_1_23" {
		t.Errorf("first package = %q, want %q", ds.Packages[0], "go_1_23")
	}
	if ds.EnvVars["CGO_ENABLED"] != "0" {
		t.Error("CGO_ENABLED should be 0")
	}
	if ds.EnvVars["GOFLAGS"] != "-mod=mod" {
		t.Error("GOFLAGS should be -mod=mod")
	}
	if ds.ShellHook == "" {
		t.Error("ShellHook should not be empty")
	}
}

func TestGenerateDevShell_Default(t *testing.T) {
	ds := DefaultDevShell()
	out, err := GenerateDevShell(ds)
	if err != nil {
		t.Fatalf("GenerateDevShell: %v", err)
	}

	checks := []string{
		"pkgs.mkShell",
		"buildInputs",
		"go_1_23",
		"gopls",
		"golangci-lint",
		"delve",
		"shellHook",
		"CGO_ENABLED",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Errorf("devshell missing %q\n---output---\n%s", c, out)
		}
	}
}

func TestGenerateDevShell_CustomPackages(t *testing.T) {
	ds := &DevShellConfig{
		Packages: []string{"python312", "nodejs_20"},
		EnvVars:  map[string]string{"MY_VAR": "hello"},
	}
	out, err := GenerateDevShell(ds)
	if err != nil {
		t.Fatalf("GenerateDevShell: %v", err)
	}

	if !strings.Contains(out, "python312") {
		t.Error("missing python312")
	}
	if !strings.Contains(out, "nodejs_20") {
		t.Error("missing nodejs_20")
	}
	if !strings.Contains(out, `MY_VAR = "hello"`) {
		t.Error("missing custom env var")
	}
}

func TestGenerateDevShell_NilConfig(t *testing.T) {
	_, err := GenerateDevShell(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestGenerateDevShell_EmptyPackages(t *testing.T) {
	_, err := GenerateDevShell(&DevShellConfig{})
	if err == nil {
		t.Error("expected error for empty packages")
	}
}

func TestGenerateDevShell_NoEnvVars(t *testing.T) {
	ds := &DevShellConfig{
		Packages: []string{"go_1_23"},
	}
	out, err := GenerateDevShell(ds)
	if err != nil {
		t.Fatalf("GenerateDevShell: %v", err)
	}
	if !strings.Contains(out, "go_1_23") {
		t.Error("missing go_1_23")
	}
}

// ---------------------------------------------------------------------------
// Build validation tests
// ---------------------------------------------------------------------------

func TestValidateBinary_Exists(t *testing.T) {
	// Create a temporary "binary" file with executable permission.
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "prompt-pulse")
	os.WriteFile(binPath, []byte("#!/bin/sh\necho hello\n"), 0o755)

	result, err := npValidateBinary(binPath)
	if err != nil {
		t.Fatalf("npValidateBinary: %v", err)
	}
	if result.BinarySize == 0 {
		t.Error("expected nonzero binary size")
	}
	if result.BinaryPath != binPath {
		t.Errorf("BinaryPath = %q, want %q", result.BinaryPath, binPath)
	}
}

func TestValidateBinary_NotExists(t *testing.T) {
	result, err := npValidateBinary("/nonexistent/binary")
	if err != nil {
		t.Fatalf("npValidateBinary should not return error, got: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for missing binary")
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for missing binary")
	}
}

func TestValidateBinary_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "empty")
	os.WriteFile(binPath, []byte{}, 0o755)

	result, err := npValidateBinary(binPath)
	if err != nil {
		t.Fatalf("npValidateBinary: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for empty binary")
	}
}

func TestValidateBinary_NotExecutable(t *testing.T) {
	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "noexec")
	os.WriteFile(binPath, []byte("content"), 0o644)

	result, err := npValidateBinary(binPath)
	if err != nil {
		t.Fatalf("npValidateBinary: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for non-executable file")
	}
}

func TestValidateBinary_Directory(t *testing.T) {
	tmp := t.TempDir()
	result, err := npValidateBinary(tmp)
	if err != nil {
		t.Fatalf("npValidateBinary: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for directory")
	}
}

func TestValidateCompletions_AllPresent(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "prompt-pulse.bash"), []byte("# bash\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "_prompt-pulse"), []byte("# zsh\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "prompt-pulse.fish"), []byte("# fish\n"), 0o644)

	err := npValidateCompletions(tmp)
	if err != nil {
		t.Errorf("npValidateCompletions: %v", err)
	}
}

func TestValidateCompletions_Missing(t *testing.T) {
	tmp := t.TempDir()
	err := npValidateCompletions(tmp)
	if err == nil {
		t.Error("expected error for missing completions")
	}
	if !strings.Contains(err.Error(), "missing completions") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Dependency checking tests
// ---------------------------------------------------------------------------

func TestCheckDeps(t *testing.T) {
	deps, err := npCheckDeps(filepath.Join("testdata", "go.mod"))
	if err != nil {
		t.Fatalf("npCheckDeps: %v", err)
	}

	// The test go.mod has 3 direct deps.
	if len(deps) != 3 {
		t.Errorf("direct deps = %d, want 3: %v", len(deps), deps)
	}

	wantDeps := []string{
		"github.com/charmbracelet/bubbletea",
		"github.com/shirou/gopsutil/v4",
		"golang.org/x/sys",
	}
	for _, want := range wantDeps {
		found := false
		for _, d := range deps {
			if d == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected dep %q in %v", want, deps)
		}
	}
}

func TestCheckDeps_MissingFile(t *testing.T) {
	_, err := npCheckDeps("/nonexistent/go.mod")
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
}

func TestParseGoVersion(t *testing.T) {
	v, err := npParseGoVersion(filepath.Join("testdata", "go.mod"))
	if err != nil {
		t.Fatalf("npParseGoVersion: %v", err)
	}
	if v != "1.23.4" {
		t.Errorf("Go version = %q, want %q", v, "1.23.4")
	}
}

func TestParseGoVersion_Missing(t *testing.T) {
	v, err := npParseGoVersion(filepath.Join("testdata", "gomod_noversion"))
	if err == nil {
		t.Errorf("expected error for missing go directive, got version %q", v)
	}
}

// ---------------------------------------------------------------------------
// Architecture detection tests
// ---------------------------------------------------------------------------

func TestParseFileOutput_Mach0ARM64(t *testing.T) {
	out := "/path/to/binary: Mach-O 64-bit executable arm64"
	arch, err := npParseFileOutput(out)
	if err != nil {
		t.Fatalf("npParseFileOutput: %v", err)
	}
	if arch != "aarch64-darwin" {
		t.Errorf("arch = %q, want %q", arch, "aarch64-darwin")
	}
}

func TestParseFileOutput_ELFX86_64(t *testing.T) {
	out := "/path/to/binary: ELF 64-bit LSB pie executable, x86-64, version 1 (SYSV)"
	arch, err := npParseFileOutput(out)
	if err != nil {
		t.Fatalf("npParseFileOutput: %v", err)
	}
	if arch != "x86_64-linux" {
		t.Errorf("arch = %q, want %q", arch, "x86_64-linux")
	}
}

func TestParseFileOutput_ELFAarch64(t *testing.T) {
	out := "/path/to/binary: ELF 64-bit LSB pie executable, ARM aarch64, version 1 (SYSV)"
	arch, err := npParseFileOutput(out)
	if err != nil {
		t.Fatalf("npParseFileOutput: %v", err)
	}
	if arch != "aarch64-linux" {
		t.Errorf("arch = %q, want %q", arch, "aarch64-linux")
	}
}

func TestParseFileOutput_Unknown(t *testing.T) {
	out := "/path/to/file: ASCII text"
	_, err := npParseFileOutput(out)
	if err == nil {
		t.Error("expected error for unknown architecture")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestGenerateDerivation_Ldflags(t *testing.T) {
	meta := DefaultMeta()
	out, err := GenerateDerivation(meta, &VendorInfo{Hash: "test"})
	if err != nil {
		t.Fatalf("GenerateDerivation: %v", err)
	}
	if !strings.Contains(out, `"-s" "-w"`) {
		t.Error("missing strip ldflags")
	}
	if !strings.Contains(out, "-X main.version") {
		t.Error("missing version ldflag")
	}
}

func TestGenerateDevShell_BalancedBraces(t *testing.T) {
	ds := DefaultDevShell()
	out, err := GenerateDevShell(ds)
	if err != nil {
		t.Fatalf("GenerateDevShell: %v", err)
	}

	opens := strings.Count(out, "{")
	closes := strings.Count(out, "}")
	if opens != closes {
		t.Errorf("unbalanced braces in devshell: %d opens, %d closes\n%s", opens, closes, out)
	}
}

func TestGenerateOverlay_BalancedBraces(t *testing.T) {
	meta := DefaultMeta()
	out, err := GenerateOverlay(meta)
	if err != nil {
		t.Fatalf("GenerateOverlay: %v", err)
	}

	opens := strings.Count(out, "{")
	closes := strings.Count(out, "}")
	if opens != closes {
		t.Errorf("unbalanced braces in overlay: %d opens, %d closes", opens, closes)
	}
}

func TestEmptyGoSum(t *testing.T) {
	tmp := t.TempDir()
	goSumPath := filepath.Join(tmp, "go.sum")
	os.WriteFile(goSumPath, []byte(""), 0o644)

	count, err := npCountDeps(goSumPath)
	if err != nil {
		t.Fatalf("npCountDeps: %v", err)
	}
	if count != 0 {
		t.Errorf("npCountDeps = %d, want 0 for empty go.sum", count)
	}
}

func TestMalformedGoSum(t *testing.T) {
	tmp := t.TempDir()
	goSumPath := filepath.Join(tmp, "go.sum")
	os.WriteFile(goSumPath, []byte("this is not a valid go.sum line\nanother garbage line\n"), 0o644)

	count, err := npCountDeps(goSumPath)
	if err != nil {
		t.Fatalf("npCountDeps: %v", err)
	}
	// Still counts non-empty lines.
	if count != 2 {
		t.Errorf("npCountDeps = %d, want 2 for 2 non-empty lines", count)
	}
}
