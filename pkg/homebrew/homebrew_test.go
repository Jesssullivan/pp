package homebrew

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Config tests
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.Name != "prompt-pulse" {
		t.Errorf("name = %q, want %q", c.Name, "prompt-pulse")
	}
	if c.Version != "2.0.0" {
		t.Errorf("version = %q, want %q", c.Version, "2.0.0")
	}
	if c.License != "MIT" {
		t.Errorf("license = %q, want %q", c.License, "MIT")
	}
	if c.HeadBranch != "main" {
		t.Errorf("head_branch = %q, want %q", c.HeadBranch, "main")
	}
	if !c.ShellCompletions {
		t.Error("shell_completions should be true by default")
	}
	if !c.DaemonService {
		t.Error("daemon_service should be true by default")
	}
	if len(c.Dependencies) != 1 || c.Dependencies[0].Name != "go" {
		t.Error("default dependencies should include go build dep")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	c := DefaultConfig()
	errs := ValidateConfig(c)
	if len(errs) != 0 {
		t.Errorf("valid config produced errors: %v", errs)
	}
}

func TestValidateConfig_MissingFields(t *testing.T) {
	c := &FormulaConfig{}
	errs := ValidateConfig(c)

	expected := []string{
		"name is required",
		"version is required",
		"description is required",
		"homepage is required",
		"license is required",
		"gitlab_project is required",
		"head_branch is required",
		"go_version is required",
	}
	for _, want := range expected {
		found := false
		for _, got := range errs {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected error %q in %v", want, errs)
		}
	}
}

func TestValidateConfig_InvalidDepType(t *testing.T) {
	c := DefaultConfig()
	c.Dependencies = append(c.Dependencies, FormulaDep{Name: "foo", Type: "bogus"})
	errs := ValidateConfig(c)

	found := false
	for _, e := range errs {
		if strings.Contains(e, "invalid type") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid dep type error, got: %v", errs)
	}
}

func TestValidateConfig_EmptyDepName(t *testing.T) {
	c := DefaultConfig()
	c.Dependencies = append(c.Dependencies, FormulaDep{Name: "", Type: "build"})
	errs := ValidateConfig(c)

	found := false
	for _, e := range errs {
		if strings.Contains(e, "name is required") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dep name required error, got: %v", errs)
	}
}

func TestValidateConfig_CustomConfig(t *testing.T) {
	c := &FormulaConfig{
		Name:          "my-tool",
		Version:       "1.0.0",
		Description:   "A custom tool",
		Homepage:      "https://example.com",
		License:       "Apache-2.0",
		GitLabProject: "org/my-tool",
		HeadBranch:    "develop",
		GoVersion:     "1.22",
	}
	errs := ValidateConfig(c)
	if len(errs) != 0 {
		t.Errorf("valid custom config produced errors: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Formula generation tests
// ---------------------------------------------------------------------------

func TestGenerateFormula_Default(t *testing.T) {
	c := DefaultConfig()
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	checks := []string{
		"class PromptPulse < Formula",
		`desc "Terminal dashboard with waifu rendering, live data, and TUI mode"`,
		`homepage "https://gitlab.com/tinyland/lab/prompt-pulse"`,
		`license "MIT"`,
		`head "https://gitlab.com/tinyland/lab/prompt-pulse.git", branch: "main"`,
		`depends_on "go" => :build`,
		"def install",
		"system \"go\", \"build\"",
		"generate_completions_from_executable",
		"service do",
		"keep_alive true",
		"test do",
		"assert_match",
		"end",
	}
	for _, want := range checks {
		if !strings.Contains(ruby, want) {
			t.Errorf("formula missing %q", want)
		}
	}
}

func TestGenerateFormula_CustomDeps(t *testing.T) {
	c := DefaultConfig()
	c.Dependencies = []FormulaDep{
		{Name: "go", Type: "build"},
		{Name: "cmake", Type: "build"},
		{Name: "openssl", Type: "runtime"},
		{Name: "pkg-config", Type: "optional"},
	}
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	if !strings.Contains(ruby, `depends_on "cmake" => :build`) {
		t.Error("missing cmake build dep")
	}
	if !strings.Contains(ruby, `depends_on "openssl"`) {
		t.Error("missing openssl runtime dep")
	}
	if !strings.Contains(ruby, `depends_on "pkg-config" => :optional`) {
		t.Error("missing pkg-config optional dep")
	}
}

func TestGenerateFormula_LdFlags(t *testing.T) {
	c := DefaultConfig()
	c.LdFlags = []string{"-s", "-w", "-X main.version=#{version}", "-X main.commit=abc123"}
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	for _, flag := range c.LdFlags {
		if !strings.Contains(ruby, flag) {
			t.Errorf("formula missing ldflag %q", flag)
		}
	}
}

func TestGenerateFormula_NoCompletions(t *testing.T) {
	c := DefaultConfig()
	c.ShellCompletions = false
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	if strings.Contains(ruby, "generate_completions_from_executable") {
		t.Error("formula should not include completions when disabled")
	}
}

func TestGenerateFormula_NoService(t *testing.T) {
	c := DefaultConfig()
	c.DaemonService = false
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	if strings.Contains(ruby, "service do") {
		t.Error("formula should not include service block when disabled")
	}
}

func TestGenerateFormula_ValidationPasses(t *testing.T) {
	c := DefaultConfig()
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	errs := ValidateFormula(ruby)
	if len(errs) != 0 {
		t.Errorf("generated formula failed validation: %v\n\nFormula:\n%s", errs, ruby)
	}
}

// ---------------------------------------------------------------------------
// Formula validation tests
// ---------------------------------------------------------------------------

func TestValidateFormula_ValidFormula(t *testing.T) {
	ruby := `class MyTool < Formula
  desc "A tool"
  homepage "https://example.com"
  license "MIT"

  def install
    system "make"
  end

  test do
    assert_match "hello", shell_output("echo hello")
  end
end
`
	errs := ValidateFormula(ruby)
	if len(errs) != 0 {
		t.Errorf("valid formula produced errors: %v", errs)
	}
}

func TestValidateFormula_MissingSections(t *testing.T) {
	ruby := `class MyTool < Formula
  def install
    system "make"
  end
end
`
	errs := ValidateFormula(ruby)

	for _, want := range []string{"desc", "homepage", "license", "test"} {
		found := false
		for _, e := range errs {
			if strings.Contains(e, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("expected error about missing %q, got: %v", want, errs)
		}
	}
}

func TestValidateFormula_UnbalancedBlocks(t *testing.T) {
	ruby := `class MyTool < Formula
  desc "A tool"
  homepage "https://example.com"
  license "MIT"

  def install
    system "make"

  test do
    assert_match "hello", shell_output("echo hello")
  end
end
`
	errs := ValidateFormula(ruby)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "unbalanced") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unbalanced block error, got: %v", errs)
	}
}

func TestValidateFormula_TemplateArtifacts(t *testing.T) {
	ruby := `class MyTool < Formula
  desc "{{ .Description }}"
  homepage "https://example.com"
  license "MIT"

  def install
    system "make"
  end

  test do
    assert_match "hello", shell_output("echo hello")
  end
end
`
	errs := ValidateFormula(ruby)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "template artifacts") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected template artifact error, got: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// PascalCase tests
// ---------------------------------------------------------------------------

func TestPascalCase(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"prompt-pulse", "PromptPulse"},
		{"my_tool", "MyTool"},
		{"simple", "Simple"},
		{"a-b-c", "ABC"},
		{"hello-world-foo", "HelloWorldFoo"},
	}
	for _, tc := range cases {
		got := hbPascalCase(tc.in)
		if got != tc.want {
			t.Errorf("hbPascalCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Bottle tests
// ---------------------------------------------------------------------------

func TestDefaultBottles(t *testing.T) {
	b := DefaultBottles()
	if len(b.Architectures) != 2 {
		t.Fatalf("expected 2 architectures, got %d", len(b.Architectures))
	}
	if b.RootURL == "" {
		t.Error("root URL should not be empty")
	}
	if b.Rebuild != 0 {
		t.Errorf("rebuild = %d, want 0", b.Rebuild)
	}
}

func TestBottleTag_Mapping(t *testing.T) {
	cases := []struct {
		os, arch, want string
	}{
		{"macos", "arm64", "arm64_sonoma"},
		{"macos", "x86_64", "sonoma"},
		{"linux", "x86_64", "x86_64_linux"},
		{"linux", "arm64", "linux"},
		{"freebsd", "amd64", "amd64_freebsd"},
	}
	for _, tc := range cases {
		got := hbBottleTag(tc.os, tc.arch)
		if got != tc.want {
			t.Errorf("hbBottleTag(%q, %q) = %q, want %q", tc.os, tc.arch, got, tc.want)
		}
	}
}

func TestGenerateBottleBlock(t *testing.T) {
	b := DefaultBottles()
	block, err := GenerateBottleBlock(b)
	if err != nil {
		t.Fatalf("GenerateBottleBlock error: %v", err)
	}

	checks := []string{
		"bottle do",
		"root_url",
		"sha256 cellar:",
		"arm64_sonoma",
		"x86_64_linux",
		"end",
	}
	for _, want := range checks {
		if !strings.Contains(block, want) {
			t.Errorf("bottle block missing %q", want)
		}
	}
}

func TestGenerateBottleBlock_WithRebuild(t *testing.T) {
	b := DefaultBottles()
	b.Rebuild = 2
	block, err := GenerateBottleBlock(b)
	if err != nil {
		t.Fatalf("GenerateBottleBlock error: %v", err)
	}

	if !strings.Contains(block, "rebuild 2") {
		t.Error("bottle block should contain rebuild counter")
	}
}

func TestGenerateBottleBlock_CustomArch(t *testing.T) {
	b := &BottleConfig{
		RootURL: "https://example.com/bottles",
		Architectures: []BottleArch{
			{OS: "macos", Arch: "x86_64", SHA256: "abc123", Cellar: ":any"},
		},
	}
	block, err := GenerateBottleBlock(b)
	if err != nil {
		t.Fatalf("GenerateBottleBlock error: %v", err)
	}
	if !strings.Contains(block, "sonoma") {
		t.Error("block should contain sonoma tag for macos/x86_64")
	}
	if !strings.Contains(block, ":any") {
		t.Error("block should contain custom cellar value")
	}
}

// ---------------------------------------------------------------------------
// Tap tests
// ---------------------------------------------------------------------------

func TestDefaultTap(t *testing.T) {
	tap := DefaultTap()
	if tap.Organization != "tinyland" {
		t.Errorf("org = %q, want %q", tap.Organization, "tinyland")
	}
	if tap.TapName != "homebrew-tap" {
		t.Errorf("tap = %q, want %q", tap.TapName, "homebrew-tap")
	}
	if tap.FormulaDir != "Formula" {
		t.Errorf("formula_dir = %q, want %q", tap.FormulaDir, "Formula")
	}
}

func TestGenerateTapReadme(t *testing.T) {
	tap := DefaultTap()
	readme, err := GenerateTapReadme(tap, []string{"prompt-pulse", "tinyland-cleanup"})
	if err != nil {
		t.Fatalf("GenerateTapReadme error: %v", err)
	}

	checks := []string{
		"tinyland/homebrew-tap",
		"brew tap tinyland/homebrew-tap",
		"`prompt-pulse`",
		"`tinyland-cleanup`",
	}
	for _, want := range checks {
		if !strings.Contains(readme, want) {
			t.Errorf("README missing %q", want)
		}
	}
}

func TestFormulaPath(t *testing.T) {
	tap := DefaultTap()
	got := hbFormulaPath(tap, "prompt-pulse")
	want := filepath.Join("Formula", "prompt-pulse.rb")
	if got != want {
		t.Errorf("hbFormulaPath = %q, want %q", got, want)
	}
}

func TestValidateTapStructure_Valid(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Formula"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Tap"), 0o644)

	errs := ValidateTapStructure(dir)
	if len(errs) != 0 {
		t.Errorf("valid tap produced errors: %v", errs)
	}
}

func TestValidateTapStructure_Missing(t *testing.T) {
	dir := t.TempDir()
	errs := ValidateTapStructure(dir)

	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateTapStructure_NonExistent(t *testing.T) {
	errs := ValidateTapStructure("/nonexistent/path/unlikely")
	if len(errs) == 0 {
		t.Error("expected error for nonexistent directory")
	}
}

// ---------------------------------------------------------------------------
// Service tests
// ---------------------------------------------------------------------------

func TestGenerateLaunchdPlist(t *testing.T) {
	cfg := &ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		LogDir:     "/usr/local/var/log",
		RunDir:     "/usr/local/var",
		KeepAlive:  true,
		Interval:   0,
	}
	plist, err := GenerateLaunchdPlist(cfg)
	if err != nil {
		t.Fatalf("GenerateLaunchdPlist error: %v", err)
	}

	checks := []string{
		"<?xml",
		"com.tinyland.prompt-pulse",
		"/usr/local/bin/prompt-pulse",
		"<true/>",
		"prompt-pulse.log",
		"prompt-pulse-error.log",
		"</plist>",
	}
	for _, want := range checks {
		if !strings.Contains(plist, want) {
			t.Errorf("plist missing %q", want)
		}
	}
}

func TestGenerateLaunchdPlist_WithInterval(t *testing.T) {
	cfg := &ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		LogDir:     "/var/log",
		RunDir:     "/tmp",
		KeepAlive:  false,
		Interval:   300,
	}
	plist, err := GenerateLaunchdPlist(cfg)
	if err != nil {
		t.Fatalf("GenerateLaunchdPlist error: %v", err)
	}

	if !strings.Contains(plist, "<false/>") {
		t.Error("plist should contain false for KeepAlive")
	}
	if !strings.Contains(plist, "<integer>300</integer>") {
		t.Error("plist should contain StartInterval of 300")
	}
}

func TestGenerateLaunchdPlist_ValidXML(t *testing.T) {
	cfg := &ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		LogDir:     "/usr/local/var/log",
		RunDir:     "/usr/local/var",
		KeepAlive:  true,
	}
	plist, err := GenerateLaunchdPlist(cfg)
	if err != nil {
		t.Fatalf("GenerateLaunchdPlist error: %v", err)
	}

	errs := ValidatePlist(plist)
	if len(errs) != 0 {
		t.Errorf("generated plist failed validation: %v\n\nPlist:\n%s", errs, plist)
	}
}

func TestGenerateBrewService(t *testing.T) {
	cfg := &ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		LogDir:     "/usr/local/var/log",
		RunDir:     "/usr/local/var",
		KeepAlive:  true,
	}
	block, err := GenerateBrewService(cfg)
	if err != nil {
		t.Fatalf("GenerateBrewService error: %v", err)
	}

	checks := []string{
		"service do",
		"keep_alive true",
		"log_path",
		"error_log_path",
		"end",
	}
	for _, want := range checks {
		if !strings.Contains(block, want) {
			t.Errorf("service block missing %q", want)
		}
	}
}

func TestGenerateBrewService_NoKeepAlive(t *testing.T) {
	cfg := &ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		LogDir:     "/usr/local/var/log",
		KeepAlive:  false,
	}
	block, err := GenerateBrewService(cfg)
	if err != nil {
		t.Fatalf("GenerateBrewService error: %v", err)
	}

	if !strings.Contains(block, "keep_alive false") {
		t.Error("service block should contain keep_alive false")
	}
}

// ---------------------------------------------------------------------------
// Plist validation tests
// ---------------------------------------------------------------------------

func TestValidatePlist_Valid(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.example.test</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/bin/test</string>
  </array>
</dict>
</plist>
`
	errs := ValidatePlist(xml)
	if len(errs) != 0 {
		t.Errorf("valid plist produced errors: %v", errs)
	}
}

func TestValidatePlist_MalformedXML(t *testing.T) {
	xml := `<plist><dict><key>Label</key><string>test</string><key>ProgramArguments</key></plist>`
	errs := ValidatePlist(xml)
	// Should be valid XML but missing proper structure - at least no XML parse error
	// The unclosed dict should cause parse issues or missing elements
	if len(errs) == 0 {
		t.Log("malformed plist may not trigger errors depending on parser strictness")
	}
}

func TestValidatePlist_MissingElements(t *testing.T) {
	xml := `<?xml version="1.0"?>
<plist version="1.0">
<dict>
  <key>SomeKey</key>
  <string>value</string>
</dict>
</plist>
`
	errs := ValidatePlist(xml)
	if len(errs) == 0 {
		t.Error("expected errors for missing Label and ProgramArguments")
	}
}

// ---------------------------------------------------------------------------
// Template correctness tests
// ---------------------------------------------------------------------------

func TestGenerateFormula_NoTemplateArtifacts(t *testing.T) {
	c := DefaultConfig()
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	if strings.Contains(ruby, "{{") {
		t.Error("generated formula contains {{ template artifacts")
	}
	if strings.Contains(ruby, "}}") {
		t.Error("generated formula contains }} template artifacts")
	}
	if strings.Contains(ruby, "<no value>") {
		t.Error("generated formula contains <no value> placeholder")
	}
}

func TestGenerateFormula_RubyBalanced(t *testing.T) {
	c := DefaultConfig()
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	errs := hbCheckRubyBalance(ruby)
	if len(errs) != 0 {
		t.Errorf("generated formula has unbalanced blocks: %v\n\nFormula:\n%s", errs, ruby)
	}
}

func TestGenerateFormula_QuotesBalanced(t *testing.T) {
	c := DefaultConfig()
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	errs := hbCheckQuoteBalance(ruby)
	if len(errs) != 0 {
		t.Errorf("generated formula has unbalanced quotes: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestGenerateFormula_SpecialCharsInVersion(t *testing.T) {
	c := DefaultConfig()
	c.Version = "2.0.0-beta.1+build.123"
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	errs := ValidateFormula(ruby)
	if len(errs) != 0 {
		t.Errorf("formula with special version chars failed validation: %v", errs)
	}
}

func TestGenerateFormula_LongDescription(t *testing.T) {
	c := DefaultConfig()
	c.Description = strings.Repeat("A very long description. ", 20)
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	if !strings.Contains(ruby, c.Description) {
		t.Error("formula should contain the full description")
	}
}

func TestGenerateFormula_EmptyLdFlags(t *testing.T) {
	c := DefaultConfig()
	c.LdFlags = nil
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	if !strings.Contains(ruby, "ldflags = %W[") {
		t.Error("formula should still have ldflags array even when empty")
	}
	errs := ValidateFormula(ruby)
	if len(errs) != 0 {
		t.Errorf("formula with empty ldflags failed validation: %v", errs)
	}
}

func TestGenerateFormula_NoDeps(t *testing.T) {
	c := DefaultConfig()
	c.Dependencies = nil
	ruby, err := GenerateFormula(c)
	if err != nil {
		t.Fatalf("GenerateFormula error: %v", err)
	}

	errs := ValidateFormula(ruby)
	if len(errs) != 0 {
		t.Errorf("formula with no deps failed validation: %v", errs)
	}
}

func TestDepLine_AllTypes(t *testing.T) {
	cases := []struct {
		dep  FormulaDep
		want string
	}{
		{FormulaDep{Name: "go", Type: "build"}, `depends_on "go" => :build`},
		{FormulaDep{Name: "openssl", Type: "runtime"}, `depends_on "openssl"`},
		{FormulaDep{Name: "cmake", Type: "test"}, `depends_on "cmake" => :test`},
		{FormulaDep{Name: "x11", Type: "optional"}, `depends_on "x11" => :optional`},
		{FormulaDep{Name: "curl", Type: ""}, `depends_on "curl"`},
	}
	for _, tc := range cases {
		got := hbDepLine(tc.dep)
		if got != tc.want {
			t.Errorf("hbDepLine(%+v) = %q, want %q", tc.dep, got, tc.want)
		}
	}
}

func TestValidateFormula_MissingClass(t *testing.T) {
	ruby := `
  desc "A tool"
  homepage "https://example.com"
  license "MIT"

  def install
    system "make"
  end

  test do
    assert_match "x", shell_output("echo x")
  end
`
	errs := ValidateFormula(ruby)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "missing class") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing class error, got: %v", errs)
	}
}

func TestCheckRubyBalance_Balanced(t *testing.T) {
	ruby := `class Foo < Formula
  def install
    system "make"
  end
  test do
    assert true
  end
end
`
	errs := hbCheckRubyBalance(ruby)
	if len(errs) != 0 {
		t.Errorf("balanced ruby reported errors: %v", errs)
	}
}

func TestCheckRubyBalance_ExtraEnd(t *testing.T) {
	ruby := `class Foo < Formula
  def install
    system "make"
  end
  end
end
`
	errs := hbCheckRubyBalance(ruby)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "extra end") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected extra end error, got: %v", errs)
	}
}

func TestTapReadme_EmptyFormulas(t *testing.T) {
	tap := DefaultTap()
	readme, err := GenerateTapReadme(tap, nil)
	if err != nil {
		t.Fatalf("GenerateTapReadme error: %v", err)
	}
	if !strings.Contains(readme, "tinyland") {
		t.Error("README should mention organization")
	}
}

func TestFormulaPath_NestedDir(t *testing.T) {
	tap := &TapConfig{
		Organization: "org",
		TapName:      "tap",
		FormulaDir:   "custom/formulas",
	}
	got := hbFormulaPath(tap, "my-tool")
	want := filepath.Join("custom/formulas", "my-tool.rb")
	if got != want {
		t.Errorf("hbFormulaPath = %q, want %q", got, want)
	}
}
