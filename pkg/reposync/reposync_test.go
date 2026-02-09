package reposync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Config tests
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.SourceRepo != "gitlab.com/tinyland/lab/crush-dots" {
		t.Errorf("SourceRepo = %q, want gitlab.com/tinyland/lab/crush-dots", c.SourceRepo)
	}
	if c.SourcePath != "cmd/prompt-pulse/" {
		t.Errorf("SourcePath = %q, want cmd/prompt-pulse/", c.SourcePath)
	}
	if c.TargetRepo != "gitlab.com/tinyland/projects/prompt-pulse" {
		t.Errorf("TargetRepo = %q, want gitlab.com/tinyland/projects/prompt-pulse", c.TargetRepo)
	}
	if c.TargetBranch != "main" {
		t.Errorf("TargetBranch = %q, want main", c.TargetBranch)
	}
	if len(c.SyncPaths) == 0 {
		t.Error("SyncPaths should not be empty")
	}
	if len(c.ExcludePaths) == 0 {
		t.Error("ExcludePaths should not be empty")
	}
	if c.CITemplate == "" {
		t.Error("CITemplate should not be empty")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	c := DefaultConfig()
	errs := ValidateConfig(c)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfig_Nil(t *testing.T) {
	errs := ValidateConfig(nil)
	if len(errs) != 1 || errs[0] != "config is nil" {
		t.Errorf("expected nil config error, got: %v", errs)
	}
}

func TestValidateConfig_MissingFields(t *testing.T) {
	c := &SyncConfig{}
	errs := ValidateConfig(c)
	if len(errs) < 4 {
		t.Errorf("expected at least 4 errors for empty config, got %d: %v", len(errs), errs)
	}
	// Check specific errors are present.
	joined := strings.Join(errs, "; ")
	for _, want := range []string{"source_repo", "source_path", "target_repo", "target_branch"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected error mentioning %q in: %s", want, joined)
		}
	}
}

func TestValidateConfig_SameRepo(t *testing.T) {
	c := DefaultConfig()
	c.TargetRepo = c.SourceRepo
	errs := ValidateConfig(c)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "must differ") {
			found = true
		}
	}
	if !found {
		t.Error("expected error about source/target being the same")
	}
}

func TestValidateConfig_OverlappingPaths(t *testing.T) {
	c := DefaultConfig()
	c.ExcludePaths = append(c.ExcludePaths, "pkg/")
	errs := ValidateConfig(c)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "both sync_paths and exclude_paths") {
			found = true
		}
	}
	if !found {
		t.Error("expected overlap error")
	}
}

func TestValidateConfig_CustomConfig(t *testing.T) {
	c := &SyncConfig{
		SourceRepo:   "github.com/org/mono",
		SourcePath:   "apps/myapp/",
		TargetRepo:   "github.com/org/myapp",
		TargetBranch: "dev",
		SyncPaths:    []string{"src/", "go.mod"},
	}
	errs := ValidateConfig(c)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfig_EmptySyncPaths(t *testing.T) {
	c := DefaultConfig()
	c.SyncPaths = nil
	errs := ValidateConfig(c)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "sync_paths must not be empty") {
			found = true
		}
	}
	if !found {
		t.Error("expected empty sync_paths error")
	}
}

// ---------------------------------------------------------------------------
// CI pipeline tests
// ---------------------------------------------------------------------------

func TestGenerateSyncPipeline(t *testing.T) {
	c := DefaultConfig()
	pipeline, err := GenerateSyncPipeline(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pipeline.Stages) != 4 {
		t.Errorf("expected 4 stages, got %d", len(pipeline.Stages))
	}
	names := make([]string, len(pipeline.Stages))
	for i, s := range pipeline.Stages {
		names[i] = s.Name
	}
	expected := []string{"detect-changes", "prepare-sync", "validate-build", "push-sync"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("stage %d = %q, want %q", i, names[i], want)
		}
	}
}

func TestGenerateSyncPipeline_NilConfig(t *testing.T) {
	_, err := GenerateSyncPipeline(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestGenerateSyncPipeline_InvalidConfig(t *testing.T) {
	_, err := GenerateSyncPipeline(&SyncConfig{})
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestGenerateSyncPipeline_Variables(t *testing.T) {
	c := DefaultConfig()
	pipeline, err := GenerateSyncPipeline(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pipeline.Variables["SOURCE_REPO"] != c.SourceRepo {
		t.Errorf("SOURCE_REPO = %q, want %q", pipeline.Variables["SOURCE_REPO"], c.SourceRepo)
	}
	if pipeline.Variables["TARGET_REPO"] != c.TargetRepo {
		t.Errorf("TARGET_REPO = %q, want %q", pipeline.Variables["TARGET_REPO"], c.TargetRepo)
	}
}

func TestGenerateSyncPipeline_PushSyncOnlyMain(t *testing.T) {
	c := DefaultConfig()
	pipeline, err := GenerateSyncPipeline(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pushStage := pipeline.Stages[3]
	if pushStage.Name != "push-sync" {
		t.Fatalf("expected push-sync stage, got %q", pushStage.Name)
	}
	if len(pushStage.Only) == 0 || pushStage.Only[0] != "main" {
		t.Error("push-sync should only run on main")
	}
}

func TestRenderGitLabCI(t *testing.T) {
	c := DefaultConfig()
	pipeline, err := GenerateSyncPipeline(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yaml, err := rsRenderGitLabCI(pipeline)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(yaml, "stages:") {
		t.Error("YAML should contain stages: keyword")
	}
	if !strings.Contains(yaml, "detect-changes:") {
		t.Error("YAML should contain detect-changes job")
	}
	if !strings.Contains(yaml, "push-sync:") {
		t.Error("YAML should contain push-sync job")
	}
	if !strings.Contains(yaml, "workflow:") {
		t.Error("YAML should contain workflow rules")
	}
}

func TestRenderGitLabCI_Nil(t *testing.T) {
	_, err := rsRenderGitLabCI(nil)
	if err == nil {
		t.Error("expected error for nil pipeline")
	}
}

func TestBuildScript_DetectChanges(t *testing.T) {
	c := DefaultConfig()
	lines := rsBuildScript("detect-changes", c)
	if len(lines) == 0 {
		t.Fatal("expected non-empty script")
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "git diff") {
			found = true
		}
	}
	if !found {
		t.Error("detect-changes should contain git diff command")
	}
}

func TestBuildScript_PrepareSync(t *testing.T) {
	c := DefaultConfig()
	lines := rsBuildScript("prepare-sync", c)
	if len(lines) == 0 {
		t.Fatal("expected non-empty script")
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "mkdir") {
			found = true
		}
	}
	if !found {
		t.Error("prepare-sync should create sync_workspace")
	}
}

func TestBuildScript_ValidateBuild(t *testing.T) {
	c := DefaultConfig()
	lines := rsBuildScript("validate-build", c)
	foundBuild := false
	foundTest := false
	for _, l := range lines {
		if strings.Contains(l, "go build") {
			foundBuild = true
		}
		if strings.Contains(l, "go test") {
			foundTest = true
		}
	}
	if !foundBuild {
		t.Error("validate-build should run go build")
	}
	if !foundTest {
		t.Error("validate-build should run go test")
	}
}

func TestBuildScript_UnknownStage(t *testing.T) {
	c := DefaultConfig()
	lines := rsBuildScript("nonexistent", c)
	if len(lines) != 1 || !strings.Contains(lines[0], "Unknown stage") {
		t.Error("expected unknown stage echo")
	}
}

func TestPipelineRules(t *testing.T) {
	c := DefaultConfig()
	pipeline, _ := GenerateSyncPipeline(c)
	if len(pipeline.Rules) < 2 {
		t.Fatalf("expected at least 2 rules, got %d", len(pipeline.Rules))
	}
	if pipeline.Rules[0].When != "on_success" {
		t.Errorf("first rule when = %q, want on_success", pipeline.Rules[0].When)
	}
}

// ---------------------------------------------------------------------------
// Flake tests
// ---------------------------------------------------------------------------

func TestParseFlakeInputs(t *testing.T) {
	content := `
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    home-manager.url = "github:nix-community/home-manager";
    home-manager.flake = true;
    prompt-pulse.url = "gitlab:tinyland/projects/prompt-pulse";
    prompt-pulse.flake = false;
  };
}
`
	inputs, err := rsParseFlakeInputs(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inputs) != 3 {
		t.Fatalf("expected 3 inputs, got %d", len(inputs))
	}
	// Check prompt-pulse.
	var pp *FlakeInput
	for i := range inputs {
		if inputs[i].Name == "prompt-pulse" {
			pp = &inputs[i]
		}
	}
	if pp == nil {
		t.Fatal("prompt-pulse input not found")
	}
	if pp.Flake {
		t.Error("prompt-pulse.flake should be false")
	}
	if pp.Type != "gitlab" {
		t.Errorf("type = %q, want gitlab", pp.Type)
	}
}

func TestParseFlakeInputs_Empty(t *testing.T) {
	_, err := rsParseFlakeInputs("")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestParseFlakeInputs_NoInputs(t *testing.T) {
	content := `{ outputs = { self }: {}; }`
	inputs, err := rsParseFlakeInputs(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inputs) != 0 {
		t.Errorf("expected 0 inputs, got %d", len(inputs))
	}
}

func TestUpdateFlakeRev(t *testing.T) {
	content := `    prompt-pulse.url = "gitlab:tinyland/projects/prompt-pulse";
    prompt-pulse.rev = "abc123";`

	result, err := rsUpdateFlakeRev(content, "prompt-pulse", "def456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `"def456"`) {
		t.Error("rev should be updated to def456")
	}
	if strings.Contains(result, `"abc123"`) {
		t.Error("old rev should be replaced")
	}
}

func TestUpdateFlakeRev_InsertNew(t *testing.T) {
	content := `    prompt-pulse.url = "gitlab:tinyland/projects/prompt-pulse";`

	result, err := rsUpdateFlakeRev(content, "prompt-pulse", "newrev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `prompt-pulse.rev = "newrev"`) {
		t.Error("should insert rev line")
	}
}

func TestUpdateFlakeRev_NotFound(t *testing.T) {
	content := `    nixpkgs.url = "github:NixOS/nixpkgs";`
	_, err := rsUpdateFlakeRev(content, "prompt-pulse", "abc")
	if err == nil {
		t.Error("expected error for missing input")
	}
}

func TestUpdateFlakeRev_EmptyName(t *testing.T) {
	_, err := rsUpdateFlakeRev("content", "", "abc")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestUpdateFlakeRev_EmptyRev(t *testing.T) {
	_, err := rsUpdateFlakeRev("content", "name", "")
	if err == nil {
		t.Error("expected error for empty rev")
	}
}

func TestGenerateFlakeInput(t *testing.T) {
	c := DefaultConfig()
	input := rsGenerateFlakeInput(c)
	if input == nil {
		t.Fatal("expected non-nil input")
	}
	if input.Name != "prompt-pulse" {
		t.Errorf("Name = %q, want prompt-pulse", input.Name)
	}
	if !strings.Contains(input.URL, "gitlab:") {
		t.Errorf("URL = %q, should start with gitlab:", input.URL)
	}
	if !input.Flake {
		t.Error("Flake should be true")
	}
}

func TestGenerateFlakeInput_Nil(t *testing.T) {
	input := rsGenerateFlakeInput(nil)
	if input != nil {
		t.Error("expected nil for nil config")
	}
}

func TestValidateFlakeInput(t *testing.T) {
	input := &FlakeInput{Name: "prompt-pulse", URL: "gitlab:tinyland/projects/prompt-pulse", Type: "gitlab"}
	errs := rsValidateFlakeInput(input)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateFlakeInput_Nil(t *testing.T) {
	errs := rsValidateFlakeInput(nil)
	if len(errs) != 1 || errs[0] != "input is nil" {
		t.Errorf("expected nil error, got: %v", errs)
	}
}

func TestValidateFlakeInput_InvalidName(t *testing.T) {
	input := &FlakeInput{Name: "123-invalid", URL: "gitlab:test", Type: "gitlab"}
	errs := rsValidateFlakeInput(input)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "valid Nix identifier") {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid identifier error")
	}
}

func TestValidateFlakeInput_MissingFields(t *testing.T) {
	input := &FlakeInput{}
	errs := rsValidateFlakeInput(input)
	if len(errs) < 2 {
		t.Errorf("expected multiple errors, got %d: %v", len(errs), errs)
	}
}

func TestInferFlakeType(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"gitlab:tinyland/project", "gitlab"},
		{"github:NixOS/nixpkgs", "github"},
		{"path:./local", "path"},
		{"git+https://example.com/repo", "git"},
		{"something-else", "indirect"},
	}
	for _, tt := range tests {
		got := rsInferFlakeType(tt.url)
		if got != tt.want {
			t.Errorf("rsInferFlakeType(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Go.mod tests
// ---------------------------------------------------------------------------

func TestRewriteGoMod(t *testing.T) {
	content := `module gitlab.com/tinyland/lab/prompt-pulse

go 1.25.5

require (
	github.com/something v1.0.0
)

replace gitlab.com/tinyland/lab/prompt-pulse/display/layout => ./display/layout
`
	result, err := rsRewriteGoMod(content,
		"gitlab.com/tinyland/lab/prompt-pulse",
		"gitlab.com/tinyland/projects/prompt-pulse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "module gitlab.com/tinyland/projects/prompt-pulse") {
		t.Error("module line should be rewritten")
	}
	if !strings.Contains(result, "gitlab.com/tinyland/projects/prompt-pulse/display/layout") {
		t.Error("replace directives should be rewritten")
	}
}

func TestRewriteGoMod_Empty(t *testing.T) {
	_, err := rsRewriteGoMod("", "old", "new")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestRewriteGoMod_EmptyOld(t *testing.T) {
	_, err := rsRewriteGoMod("module foo\n", "", "new")
	if err == nil {
		t.Error("expected error for empty old module")
	}
}

func TestRewriteGoMod_EmptyNew(t *testing.T) {
	_, err := rsRewriteGoMod("module foo\n", "old", "")
	if err == nil {
		t.Error("expected error for empty new module")
	}
}

func TestRewriteGoMod_SameModule(t *testing.T) {
	content := "module foo\ngo 1.21\n"
	result, err := rsRewriteGoMod(content, "foo", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Error("same module should return unchanged content")
	}
}

func TestRewriteGoMod_NotFound(t *testing.T) {
	_, err := rsRewriteGoMod("module other\ngo 1.21\n", "missing", "new")
	if err == nil {
		t.Error("expected error when module not found")
	}
}

func TestRewriteImports(t *testing.T) {
	goFile := `package main

import (
	"fmt"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/reposync"
	"gitlab.com/tinyland/lab/prompt-pulse/internal/config"
)

func main() {
	fmt.Println("hello")
}
`
	result, err := rsRewriteImports(goFile,
		"gitlab.com/tinyland/lab/prompt-pulse",
		"gitlab.com/tinyland/projects/prompt-pulse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "gitlab.com/tinyland/lab/prompt-pulse") {
		t.Error("old import paths should be replaced")
	}
	if !strings.Contains(result, "gitlab.com/tinyland/projects/prompt-pulse/pkg/reposync") {
		t.Error("new import path should be present")
	}
}

func TestRewriteImports_EmptyOld(t *testing.T) {
	_, err := rsRewriteImports("content", "", "new")
	if err == nil {
		t.Error("expected error for empty old module")
	}
}

func TestRewriteImports_SameModule(t *testing.T) {
	content := `import "foo/bar"`
	result, err := rsRewriteImports(content, "foo", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Error("same module should return unchanged content")
	}
}

func TestDetectModule(t *testing.T) {
	content := "module gitlab.com/tinyland/lab/prompt-pulse\n\ngo 1.25.5\n"
	mod, err := rsDetectModule(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mod != "gitlab.com/tinyland/lab/prompt-pulse" {
		t.Errorf("module = %q, want gitlab.com/tinyland/lab/prompt-pulse", mod)
	}
}

func TestDetectModule_Empty(t *testing.T) {
	_, err := rsDetectModule("")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestDetectModule_NoDeclaration(t *testing.T) {
	_, err := rsDetectModule("go 1.21\n")
	if err == nil {
		t.Error("expected error for missing module line")
	}
}

func TestValidateGoMod(t *testing.T) {
	content := "module foo\n\ngo 1.25\n"
	issues := rsValidateGoMod(content)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %v", issues)
	}
}

func TestValidateGoMod_Empty(t *testing.T) {
	issues := rsValidateGoMod("")
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got: %v", issues)
	}
}

func TestValidateGoMod_MissingModule(t *testing.T) {
	issues := rsValidateGoMod("go 1.21\n")
	found := false
	for _, i := range issues {
		if strings.Contains(i, "missing module") {
			found = true
		}
	}
	if !found {
		t.Error("expected missing module warning")
	}
}

func TestValidateGoMod_MissingGoVersion(t *testing.T) {
	issues := rsValidateGoMod("module foo\n")
	found := false
	for _, i := range issues {
		if strings.Contains(i, "missing go version") {
			found = true
		}
	}
	if !found {
		t.Error("expected missing go version warning")
	}
}

func TestValidateGoMod_ReplaceDirectives(t *testing.T) {
	content := "module foo\ngo 1.21\nreplace foo/bar => ./bar\n"
	issues := rsValidateGoMod(content)
	if len(issues) < 1 {
		t.Error("expected warnings about replace directives")
	}
}

// ---------------------------------------------------------------------------
// Manifest tests
// ---------------------------------------------------------------------------

func TestGenerateManifest(t *testing.T) {
	// Create a temp directory with mock files.
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module test\ngo 1.21\n")
	writeFile(t, dir, "main.go", "package main\nfunc main(){}\n")
	writeFile(t, dir, "pkg/foo/foo.go", "package foo\n")
	writeFile(t, dir, "internal/bar.go", "package bar\n")
	writeFile(t, dir, "docs/README.md", "# Docs\n")

	config := &SyncConfig{
		SyncPaths:    []string{"pkg/", "go.mod", "main.go", "*.go", "docs/"},
		ExcludePaths: []string{"internal/"},
	}

	manifest, err := rsGenerateManifest(config, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manifest.Version != "1" {
		t.Errorf("version = %q, want 1", manifest.Version)
	}
	if len(manifest.Files) == 0 {
		t.Fatal("expected files in manifest")
	}

	// Check internal/bar.go is excluded.
	for _, f := range manifest.Files {
		if f.SourcePath == "internal/bar.go" && f.Action != "exclude" {
			t.Error("internal/bar.go should be excluded")
		}
	}

	// Check go.mod is marked as rewrite.
	for _, f := range manifest.Files {
		if f.SourcePath == "go.mod" && f.Action != "rewrite" {
			t.Errorf("go.mod action = %q, want rewrite", f.Action)
		}
	}
}

func TestGenerateManifest_NilConfig(t *testing.T) {
	_, err := rsGenerateManifest(nil, ".")
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestGenerateManifest_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	config := &SyncConfig{SyncPaths: []string{"*.go"}}
	manifest, err := rsGenerateManifest(config, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifest.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(manifest.Files))
	}
}

func TestRenderManifestJSON(t *testing.T) {
	manifest := &SyncManifest{
		Version:      "1",
		SourceCommit: "abc123",
		Files: []SyncFile{
			{SourcePath: "main.go", TargetPath: "main.go", Hash: "deadbeef", Action: "rewrite"},
		},
		GeneratedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	out, err := rsRenderManifestJSON(manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify it's valid JSON.
	var parsed SyncManifest
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.Version != "1" {
		t.Error("parsed version should be 1")
	}
	if len(parsed.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(parsed.Files))
	}
}

func TestRenderManifestJSON_Nil(t *testing.T) {
	_, err := rsRenderManifestJSON(nil)
	if err == nil {
		t.Error("expected error for nil manifest")
	}
}

func TestRenderManifestMarkdown(t *testing.T) {
	manifest := &SyncManifest{
		Version:      "1",
		SourceCommit: "abc123",
		Files: []SyncFile{
			{SourcePath: "main.go", TargetPath: "main.go", Hash: "aabbccddee", Action: "rewrite"},
			{SourcePath: "internal/x.go", TargetPath: "internal/x.go", Action: "exclude"},
		},
		GeneratedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	md := rsRenderManifestMarkdown(manifest)
	if !strings.Contains(md, "# Sync Manifest") {
		t.Error("should contain heading")
	}
	if !strings.Contains(md, "abc123") {
		t.Error("should contain source commit")
	}
	if !strings.Contains(md, "rewrite") {
		t.Error("should list rewrite action")
	}
	if !strings.Contains(md, "exclude") {
		t.Error("should list exclude action")
	}
}

func TestRenderManifestMarkdown_Nil(t *testing.T) {
	md := rsRenderManifestMarkdown(nil)
	if !strings.Contains(md, "No manifest available") {
		t.Error("nil manifest should produce fallback message")
	}
}

// ---------------------------------------------------------------------------
// Path filtering tests
// ---------------------------------------------------------------------------

func TestFilterPaths_IncludeOnly(t *testing.T) {
	all := []string{"pkg/a.go", "main.go", "internal/b.go", "docs/c.md"}
	result := rsFilterPaths(all, []string{"pkg/", "*.go"}, nil)
	if len(result) != 3 {
		t.Errorf("expected 3 paths, got %d: %v", len(result), result)
	}
}

func TestFilterPaths_ExcludeOnly(t *testing.T) {
	all := []string{"pkg/a.go", "internal/b.go", "docs/c.md"}
	// With no includes, everything is excluded by default.
	result := rsFilterPaths(all, nil, []string{"internal/"})
	if len(result) != 0 {
		t.Errorf("expected 0 paths (no includes), got %d: %v", len(result), result)
	}
}

func TestFilterPaths_Combined(t *testing.T) {
	all := []string{"pkg/a.go", "pkg/b.go", "internal/c.go", "main.go", "vendor/d.go"}
	result := rsFilterPaths(all,
		[]string{"pkg/", "*.go", "vendor/"},
		[]string{"internal/"})
	// pkg/a.go, pkg/b.go, main.go, vendor/d.go should pass; internal/c.go excluded.
	if len(result) != 4 {
		t.Errorf("expected 4 paths, got %d: %v", len(result), result)
	}
	for _, p := range result {
		if strings.HasPrefix(p, "internal/") {
			t.Errorf("internal/ path should be excluded: %s", p)
		}
	}
}

func TestFilterPaths_GlobPatterns(t *testing.T) {
	all := []string{"main.go", "util.go", "readme.md", "config.toml"}
	result := rsFilterPaths(all, []string{"*.go"}, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 .go files, got %d: %v", len(result), result)
	}
}

func TestFilterPaths_EmptyLists(t *testing.T) {
	all := []string{"a.go", "b.go"}
	result := rsFilterPaths(all, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 paths with no includes, got %d", len(result))
	}
}

func TestFilterPaths_EmptyInput(t *testing.T) {
	result := rsFilterPaths(nil, []string{"*.go"}, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 paths from nil input, got %d", len(result))
	}
}

func TestPathMatch_DirectoryPrefix(t *testing.T) {
	if !rsPathMatch("pkg/foo/bar.go", "pkg/") {
		t.Error("should match directory prefix")
	}
	if rsPathMatch("notpkg/bar.go", "pkg/") {
		t.Error("should not match different prefix")
	}
}

func TestPathMatch_Extension(t *testing.T) {
	if !rsPathMatch("main.go", "*.go") {
		t.Error("should match extension")
	}
	if rsPathMatch("main.rs", "*.go") {
		t.Error("should not match wrong extension")
	}
}

func TestPathMatch_Exact(t *testing.T) {
	if !rsPathMatch("go.mod", "go.mod") {
		t.Error("should match exact")
	}
	if rsPathMatch("go.sum", "go.mod") {
		t.Error("should not match different file")
	}
}

// ---------------------------------------------------------------------------
// Nightly tests
// ---------------------------------------------------------------------------

func TestDefaultNightly(t *testing.T) {
	n := DefaultNightly()
	if n.Schedule != "0 3 * * *" {
		t.Errorf("Schedule = %q, want '0 3 * * *'", n.Schedule)
	}
	if len(n.FlakeInputs) != 1 || n.FlakeInputs[0] != "prompt-pulse" {
		t.Errorf("FlakeInputs = %v, want [prompt-pulse]", n.FlakeInputs)
	}
	if n.AutoMerge {
		t.Error("AutoMerge should be false by default")
	}
	if n.BranchPrefix != "nightly/flake-update" {
		t.Errorf("BranchPrefix = %q, want nightly/flake-update", n.BranchPrefix)
	}
}

func TestGenerateNightlyJob(t *testing.T) {
	n := DefaultNightly()
	yaml, err := rsGenerateNightlyJob(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(yaml, "nightly-flake-update:") {
		t.Error("should contain job name")
	}
	if !strings.Contains(yaml, "nix flake update prompt-pulse") {
		t.Error("should contain flake update command")
	}
	if !strings.Contains(yaml, "nixos/nix:latest") {
		t.Error("should use nix image")
	}
	// Default AutoMerge is false, so no merge_request curl.
	if strings.Contains(yaml, "merge_requests") {
		t.Error("should not contain MR creation when AutoMerge=false")
	}
}

func TestGenerateNightlyJob_AutoMerge(t *testing.T) {
	n := DefaultNightly()
	n.AutoMerge = true
	yaml, err := rsGenerateNightlyJob(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(yaml, "merge_when_pipeline_succeeds") {
		t.Error("should contain merge request creation when AutoMerge=true")
	}
}

func TestGenerateNightlyJob_Nil(t *testing.T) {
	_, err := rsGenerateNightlyJob(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestGenerateNightlyJob_InvalidSchedule(t *testing.T) {
	n := DefaultNightly()
	n.Schedule = "invalid"
	_, err := rsGenerateNightlyJob(n)
	if err == nil {
		t.Error("expected error for invalid schedule")
	}
}

func TestValidateSchedule_Valid(t *testing.T) {
	cases := []string{
		"0 3 * * *",
		"*/5 * * * *",
		"0 0 1 1 0",
		"30 12 15 6 1-5",
	}
	for _, c := range cases {
		if err := rsValidateSchedule(c); err != nil {
			t.Errorf("rsValidateSchedule(%q) unexpected error: %v", c, err)
		}
	}
}

func TestValidateSchedule_Invalid(t *testing.T) {
	cases := []struct {
		cron string
		desc string
	}{
		{"", "empty"},
		{"0 3 *", "too few fields"},
		{"0 3 * * * *", "too many fields"},
		{"0 3 * * MON", "alphabetic day"},
	}
	for _, tc := range cases {
		if err := rsValidateSchedule(tc.cron); err == nil {
			t.Errorf("rsValidateSchedule(%q) [%s] expected error", tc.cron, tc.desc)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestRewriteGoMod_ComplexReplace(t *testing.T) {
	content := `module gitlab.com/tinyland/lab/prompt-pulse

go 1.25.5

replace gitlab.com/tinyland/lab/prompt-pulse/display/layout => ./display/layout

replace gitlab.com/tinyland/lab/prompt-pulse/display/render => ./display/render

replace gitlab.com/tinyland/lab/prompt-pulse/tests/mocks => ./tests/mocks
`
	result, err := rsRewriteGoMod(content,
		"gitlab.com/tinyland/lab/prompt-pulse",
		"gitlab.com/tinyland/projects/prompt-pulse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All three replace directives should be rewritten.
	count := strings.Count(result, "gitlab.com/tinyland/projects/prompt-pulse")
	if count < 4 { // 1 module + 3 replaces
		t.Errorf("expected at least 4 occurrences of new module, got %d", count)
	}
	// No old module references should remain.
	if strings.Contains(result, "gitlab.com/tinyland/lab/prompt-pulse") {
		t.Error("old module path should not remain")
	}
}

func TestFlakeInputWithRev(t *testing.T) {
	content := `
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixpkgs.rev = "abc123def456";
`
	inputs, err := rsParseFlakeInputs(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inputs) != 1 {
		t.Fatalf("expected 1 input, got %d", len(inputs))
	}
	if inputs[0].Rev != "abc123def456" {
		t.Errorf("rev = %q, want abc123def456", inputs[0].Rev)
	}
}

func TestRepoToFlakeURL(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"gitlab.com/tinyland/projects/prompt-pulse", "gitlab:tinyland/projects/prompt-pulse"},
		{"github.com/NixOS/nixpkgs", "github:NixOS/nixpkgs"},
		{"example.com/repo", "git+https://example.com/repo"},
	}
	for _, tt := range tests {
		got := rsRepoToFlakeURL(tt.repo)
		if got != tt.want {
			t.Errorf("rsRepoToFlakeURL(%q) = %q, want %q", tt.repo, got, tt.want)
		}
	}
}

func TestRepoShortName(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"gitlab.com/tinyland/projects/prompt-pulse", "prompt-pulse"},
		{"github.com/org/repo", "repo"},
		{"single", "single"},
	}
	for _, tt := range tests {
		got := rsRepoShortName(tt.repo)
		if got != tt.want {
			t.Errorf("rsRepoShortName(%q) = %q, want %q", tt.repo, got, tt.want)
		}
	}
}

func TestEscapePaths(t *testing.T) {
	paths := []string{"*.go", "go.mod", "pkg/"}
	escaped := rsEscapePaths(paths)
	if escaped[0] != `.*\.go` {
		t.Errorf("escaped[0] = %q, want '.*\\.go'", escaped[0])
	}
	if escaped[1] != `go\.mod` {
		t.Errorf("escaped[1] = %q, want 'go\\.mod'", escaped[1])
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
