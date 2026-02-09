package cleanup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- helpers ---

// clCreateTestTree builds a temporary directory tree with mock Go files for testing.
// Returns the root path and a cleanup function.
func clCreateTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// v1 directories.
	v1Files := map[string]string{
		"display/banner/layout.go": `package banner

import "fmt"

// Layout renders a banner layout.
func Layout() string {
	return fmt.Sprintf("banner layout")
}
`,
		"display/banner/terminal.go": `package banner

import "os"

// DetectTerminal returns terminal type.
func DetectTerminal() string {
	return os.Getenv("TERM")
}
`,
		"display/banner/box.go": `package banner

// Box draws a box around text.
func Box(text string) string {
	return "+" + text + "+"
}
`,
		"display/banner/layout_test.go": `package banner

import "testing"

func TestLayout(t *testing.T) {
	_ = Layout()
}
`,
		"display/color/color.go": `package color

// Theme holds color theme data.
type Theme struct {
	Name string
}
`,
		"display/render/protocol.go": `package render

// Protocol defines a rendering protocol.
type Protocol interface {
	Render(data []byte) error
}
`,
		"waifu/render.go": `package waifu

import "fmt"

// Render draws a waifu image.
func Render(path string) error {
	fmt.Println("rendering", path)
	return nil
}
`,
		"waifu/session.go": `package waifu

// Session tracks a waifu rendering session.
type Session struct {
	ID   int
	Path string
}
`,
		"collectors/models.go": `package collectors

// Metric represents a collected metric.
type Metric struct {
	Name  string
	Value float64
}
`,
		"collectors/collector.go": `package collectors

// Collector gathers system metrics.
type Collector interface {
	Collect() ([]Metric, error)
}
`,
		"shell/bash.go": `package shell

// BashIntegration provides bash shell integration.
func BashIntegration() string {
	return "eval $(prompt-pulse init bash)"
}
`,
		"config/config.go": `package config

// Config holds application configuration.
type Config struct {
	Theme    string
	CacheDir string
}
`,
		"cache/store.go": `package cache

import "sync"

// Store is a simple key-value cache.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
}
`,
		"status/evaluator.go": `package status

// Evaluate checks system status.
func Evaluate() string {
	return "ok"
}
`,
		"internal/format/strings.go": `package format

// Truncate shortens a string to n characters.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
`,
		"scripts/check-secrets.sh": `#!/bin/bash
echo "checking secrets"
`,
	}

	// v2 packages.
	v2Files := map[string]string{
		"pkg/layout/layout.go": `package layout

// Constraint represents a layout constraint.
type Constraint struct {
	Name   string
	Weight float64
}

// Solve resolves layout constraints.
func Solve(constraints []Constraint) {}
`,
		"pkg/layout/layout_test.go": `package layout

import "testing"

func TestSolve(t *testing.T) {
	Solve(nil)
}
`,
		"pkg/terminal/terminal.go": `package terminal

// Detect performs 7-layer terminal detection.
func Detect() string {
	return "detected"
}
`,
		"pkg/terminal/terminal_test.go": `package terminal

import "testing"

func TestDetect(t *testing.T) {
	_ = Detect()
}
`,
		"pkg/components/box.go": `package components

// Box renders a box component.
type Box struct {
	Width  int
	Height int
}
`,
		"pkg/image/render.go": `package image

// Renderer supports multiple image protocols.
type Renderer struct {
	Protocol string
}
`,
		"pkg/waifu/waifu.go": `package waifu

// Session manages a waifu rendering session with PID tracking.
type Session struct {
	PID  int
	Path string
}
`,
		"pkg/waifu/waifu_test.go": `package waifu

import "testing"

func TestSession(t *testing.T) {
	_ = Session{PID: 1}
}
`,
		"pkg/collectors/collector.go": `package collectors

// Collector uses duck-typed interface for data collection.
type Collector interface {
	Name() string
	Collect() error
}
`,
		"pkg/data/data.go": `package data

// TimeSeries stores SoA time-series data.
type TimeSeries struct {
	Timestamps []int64
	Values     []float64
}
`,
		"pkg/shell/shell.go": `package shell

// Shell provides 4-shell integration.
type Shell interface {
	Init() string
}
`,
		"pkg/config/config.go": `package config

// Config supports nested TOML configuration.
type Config struct {
	Theme string
	Cache CacheConfig
}

// CacheConfig holds cache settings.
type CacheConfig struct {
	Dir string
	TTL int
}
`,
		"pkg/cache/cache.go": `package cache

// LRU implements an LRU cache with atomic writes.
type LRU struct {
	MaxSize int
}
`,
		"pkg/theme/theme.go": `package theme

// Theme defines the color theme.
type Theme struct {
	Name   string
	Colors map[string]string
}
`,
		"pkg/widgets/sysmetrics.go": `package widgets

// SysMetrics is the system metrics widget.
type SysMetrics struct {
	CPUPercent float64
}
`,
		"pkg/starship/starship.go": `package starship

// Config parses starship configuration.
type Config struct {
	Format string
}
`,
		"pkg/tui/app.go": `package tui

// App is the v2 TUI application.
type App struct {
	Width  int
	Height int
}
`,
	}

	// Write all files.
	for relPath, content := range v1Files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", absPath, err)
		}
	}
	for relPath, content := range v2Files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", absPath, err)
		}
	}

	return root
}

// --- Manifest tests ---

func TestManifestCreation(t *testing.T) {
	m := &CleanupManifest{}
	if m.Deletions != nil {
		t.Error("expected nil Deletions on new manifest")
	}
	if m.Modifications != nil {
		t.Error("expected nil Modifications on new manifest")
	}
}

func TestManifestAddDeletions(t *testing.T) {
	m := &CleanupManifest{}
	m.Deletions = append(m.Deletions, Deletion{
		Path:         "display/banner",
		Reason:       "replaced by pkg/layout",
		Safe:         true,
		LinesRemoved: 100,
		ReplacedBy:   "pkg/layout/",
	})
	if len(m.Deletions) != 1 {
		t.Fatalf("expected 1 deletion, got %d", len(m.Deletions))
	}
	if m.Deletions[0].Path != "display/banner" {
		t.Errorf("unexpected path: %s", m.Deletions[0].Path)
	}
}

func TestManifestAddModifications(t *testing.T) {
	m := &CleanupManifest{}
	m.Modifications = append(m.Modifications, Modification{
		Path:        "main.go",
		Description: "update import",
		OldImport:   "prompt-pulse/display/banner",
		NewImport:   "prompt-pulse/pkg/layout",
	})
	if len(m.Modifications) != 1 {
		t.Fatalf("expected 1 modification, got %d", len(m.Modifications))
	}
}

func TestManifestSummaryCalculation(t *testing.T) {
	m := &CleanupManifest{
		Deletions: []Deletion{
			{Path: "a", Safe: true, LinesRemoved: 50},
			{Path: "b", Safe: true, LinesRemoved: 30},
			{Path: "c", Safe: false, LinesRemoved: 20},
		},
		Modifications: []Modification{
			{Path: "d"},
		},
	}
	s := clComputeSummary(m)
	if s.TotalDeletions != 3 {
		t.Errorf("expected 3 deletions, got %d", s.TotalDeletions)
	}
	if s.SafeDeletions != 2 {
		t.Errorf("expected 2 safe, got %d", s.SafeDeletions)
	}
	if s.UnsafeDeletions != 1 {
		t.Errorf("expected 1 unsafe, got %d", s.UnsafeDeletions)
	}
	if s.TotalLinesRemoved != 100 {
		t.Errorf("expected 100 lines, got %d", s.TotalLinesRemoved)
	}
	if s.TotalModifications != 1 {
		t.Errorf("expected 1 modification, got %d", s.TotalModifications)
	}
}

func TestManifestMarkdownRendering(t *testing.T) {
	m := &CleanupManifest{
		Deletions: []Deletion{
			{Path: "display/banner", Reason: "replaced", Safe: true, LinesRemoved: 100, ReplacedBy: "pkg/layout/"},
			{Path: "scripts", Reason: "manual review", Safe: false, LinesRemoved: 10},
		},
		Summary: ManifestSummary{
			TotalDeletions:  2,
			SafeDeletions:   1,
			UnsafeDeletions: 1,
			TotalLinesRemoved: 110,
		},
	}

	md := m.RenderMarkdown()
	if !strings.Contains(md, "# Cleanup Manifest") {
		t.Error("missing markdown header")
	}
	if !strings.Contains(md, "display/banner") {
		t.Error("missing deletion entry")
	}
	if !strings.Contains(md, "[SAFE]") {
		t.Error("missing SAFE label")
	}
	if !strings.Contains(md, "[UNSAFE]") {
		t.Error("missing UNSAFE label")
	}
	if !strings.Contains(md, "pkg/layout/") {
		t.Error("missing replacement reference")
	}
}

func TestManifestScriptRendering(t *testing.T) {
	m := &CleanupManifest{
		Deletions: []Deletion{
			{Path: "display/banner", Reason: "replaced", Safe: true, LinesRemoved: 100, ReplacedBy: "pkg/layout/"},
			{Path: "scripts", Reason: "no Go replacement", Safe: false, LinesRemoved: 10},
		},
		Summary: ManifestSummary{
			TotalDeletions:    2,
			SafeDeletions:     1,
			UnsafeDeletions:   1,
			TotalLinesRemoved: 110,
		},
	}

	script := m.RenderScript()
	if !strings.Contains(script, "#!/usr/bin/env bash") {
		t.Error("missing shebang")
	}
	if !strings.Contains(script, "set -euo pipefail") {
		t.Error("missing strict mode")
	}
	// Safe deletion should be an active rm command.
	if !strings.Contains(script, `rm -rf "display/banner"`) {
		t.Error("missing safe rm command")
	}
	// Unsafe deletion should be commented out.
	if !strings.Contains(script, `# rm -rf "scripts"`) {
		t.Error("unsafe deletion should be commented out")
	}
}

func TestManifestMarkdownWithModifications(t *testing.T) {
	m := &CleanupManifest{
		Modifications: []Modification{
			{Path: "main.go", OldImport: "old/import", NewImport: "new/import"},
		},
		Summary: ManifestSummary{TotalModifications: 1},
	}

	md := m.RenderMarkdown()
	if !strings.Contains(md, "Modifications") {
		t.Error("missing Modifications section")
	}
	if !strings.Contains(md, "old/import") {
		t.Error("missing old import in table")
	}
	if !strings.Contains(md, "new/import") {
		t.Error("missing new import in table")
	}
}

// --- Scanner tests ---

func TestScanDirectory(t *testing.T) {
	root := clCreateTestTree(t)
	files, err := clScanDirectory(filepath.Join(root, "display", "banner"))
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected files in display/banner")
	}
	// Should find layout.go, terminal.go, box.go, layout_test.go.
	if len(files) != 4 {
		t.Errorf("expected 4 files, got %d", len(files))
	}
}

func TestScanDirectoryNonexistent(t *testing.T) {
	_, err := clScanDirectory("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestScanDirectoryEmpty(t *testing.T) {
	empty := t.TempDir()
	files, err := clScanDirectory(empty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}

func TestParseGoFile(t *testing.T) {
	root := clCreateTestTree(t)
	fi, err := clParseGoFile(filepath.Join(root, "display", "banner", "layout.go"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if fi.Package != "banner" {
		t.Errorf("expected package banner, got %s", fi.Package)
	}
	if fi.Lines == 0 {
		t.Error("expected non-zero line count")
	}
	if fi.IsTest {
		t.Error("layout.go should not be a test file")
	}
	// Should have "fmt" import.
	found := false
	for _, imp := range fi.Imports {
		if imp == "fmt" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'fmt' in imports")
	}
	// Should have Layout as exported symbol.
	foundExport := false
	for _, sym := range fi.ExportedSymbols {
		if sym == "Layout" {
			foundExport = true
		}
	}
	if !foundExport {
		t.Errorf("expected Layout in exported symbols, got %v", fi.ExportedSymbols)
	}
}

func TestParseGoFileTestFile(t *testing.T) {
	root := clCreateTestTree(t)
	fi, err := clParseGoFile(filepath.Join(root, "display", "banner", "layout_test.go"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !fi.IsTest {
		t.Error("layout_test.go should be a test file")
	}
}

func TestCountLines(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.go")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := clCountLines(path)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if lines != 3 {
		t.Errorf("expected 3 lines, got %d", lines)
	}
}

func TestCountLinesNoTrailingNewline(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.go")
	content := "line1\nline2"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := clCountLines(path)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if lines != 2 {
		t.Errorf("expected 2 lines, got %d", lines)
	}
}

func TestCountLinesEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty.go")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := clCountLines(path)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if lines != 0 {
		t.Errorf("expected 0 lines for empty file, got %d", lines)
	}
}

func TestFindDuplicates(t *testing.T) {
	v1 := []FileInfo{
		{Path: "display/banner/layout.go", Package: "banner"},
		{Path: "waifu/render.go", Package: "waifu"},
		{Path: "collectors/models.go", Package: "collectors"},
	}
	v2 := []FileInfo{
		{Path: "pkg/layout/layout.go", Package: "layout"},
		{Path: "pkg/waifu/waifu.go", Package: "waifu"},
		{Path: "pkg/collectors/collector.go", Package: "collectors"},
		{Path: "pkg/data/data.go", Package: "data"},
	}

	dupes := clFindDuplicates(v1, v2)
	if len(dupes) == 0 {
		t.Fatal("expected at least one duplicate")
	}

	// waifu/render.go should match pkg/waifu/waifu.go by package name.
	foundWaifu := false
	for _, d := range dupes {
		if d.V1Path == "waifu/render.go" && d.V2Package == "waifu" {
			foundWaifu = true
			if d.Confidence < 0.5 {
				t.Errorf("waifu confidence too low: %f", d.Confidence)
			}
		}
	}
	if !foundWaifu {
		t.Error("expected waifu duplicate match")
	}
}

func TestFindDuplicatesExactFilename(t *testing.T) {
	v1 := []FileInfo{
		{Path: "display/banner/layout.go", Package: "banner"},
	}
	v2 := []FileInfo{
		{Path: "pkg/layout/layout.go", Package: "layout"},
	}

	dupes := clFindDuplicates(v1, v2)
	if len(dupes) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(dupes))
	}
	if dupes[0].Reason != "same filename" {
		t.Errorf("expected 'same filename' reason, got %q", dupes[0].Reason)
	}
}

func TestFindDuplicatesSkipsTests(t *testing.T) {
	v1 := []FileInfo{
		{Path: "display/banner/layout_test.go", Package: "banner", IsTest: true},
	}
	v2 := []FileInfo{
		{Path: "pkg/layout/layout_test.go", Package: "layout", IsTest: false},
	}

	dupes := clFindDuplicates(v1, v2)
	if len(dupes) != 0 {
		t.Error("expected no duplicates for test files")
	}
}

func TestFindDuplicatesEmpty(t *testing.T) {
	dupes := clFindDuplicates(nil, nil)
	if len(dupes) != 0 {
		t.Error("expected no duplicates for empty inputs")
	}
}

// --- V1 directories tests ---

func TestV1DirectoriesAllListed(t *testing.T) {
	dirs := clV1Directories()
	if len(dirs) < 10 {
		t.Errorf("expected at least 10 v1 directories, got %d", len(dirs))
	}

	// Check known directories exist.
	expected := []string{
		"display/banner", "display/color", "display/layout", "display/render",
		"display/starship", "display/tui", "display/widgets",
		"waifu", "collectors", "shell", "config", "cache",
		"status", "internal/format", "cmd/demo-mocks", "scripts",
	}
	dirMap := make(map[string]bool)
	for _, d := range dirs {
		dirMap[d.Path] = true
	}
	for _, exp := range expected {
		if !dirMap[exp] {
			t.Errorf("missing expected v1 directory: %s", exp)
		}
	}
}

func TestV1DirectoriesHaveReplacements(t *testing.T) {
	dirs := clV1Directories()
	for _, d := range dirs {
		if d.V2Replacement == "" {
			t.Errorf("v1 directory %q has no v2 replacement", d.Path)
		}
	}
}

func TestV1DirectoriesSafeFlags(t *testing.T) {
	dirs := clV1Directories()
	unsafeCount := 0
	for _, d := range dirs {
		if !d.Safe {
			unsafeCount++
		}
	}
	// scripts should be the only unsafe one.
	if unsafeCount != 1 {
		t.Errorf("expected 1 unsafe directory, got %d", unsafeCount)
	}
}

func TestV1DirectoriesDescriptions(t *testing.T) {
	dirs := clV1Directories()
	for _, d := range dirs {
		if d.Description == "" {
			t.Errorf("v1 directory %q has no description", d.Path)
		}
	}
}

// --- Import analysis tests ---

func TestFindV1Imports(t *testing.T) {
	root := t.TempDir()

	// Create a file that imports a v1 package.
	mainDir := filepath.Join(root, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainGo := filepath.Join(mainDir, "main.go")
	content := `package main

import (
	"fmt"
	"gitlab.com/tinyland/lab/prompt-pulse/display/banner"
	"gitlab.com/tinyland/lab/prompt-pulse/collectors"
)

func main() {
	fmt.Println(banner.Layout())
}
`
	if err := os.WriteFile(mainGo, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	refs, err := clFindV1Imports(root)
	if err != nil {
		t.Fatalf("find imports failed: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 v1 import refs, got %d", len(refs))
	}

	// Check that replacements are suggested.
	for _, ref := range refs {
		if ref.V2Replacement == "" {
			t.Errorf("no replacement for import %s", ref.ImportPath)
		}
	}
}

func TestSuggestImportUpdate(t *testing.T) {
	ref := ImportRef{
		ImportPath:    "gitlab.com/tinyland/lab/prompt-pulse/display/banner",
		V2Replacement: "gitlab.com/tinyland/lab/prompt-pulse/pkg/layout",
	}
	suggestion := clSuggestImportUpdate(ref)
	if !strings.Contains(suggestion, "pkg/layout") {
		t.Errorf("suggestion should contain pkg/layout: %s", suggestion)
	}
	if !strings.Contains(suggestion, "was") {
		t.Errorf("suggestion should reference old import: %s", suggestion)
	}
}

func TestSuggestImportUpdateNoReplacement(t *testing.T) {
	ref := ImportRef{ImportPath: "some/package"}
	suggestion := clSuggestImportUpdate(ref)
	if suggestion != "no v2 replacement available" {
		t.Errorf("unexpected suggestion: %s", suggestion)
	}
}

func TestFindOrphanedImports(t *testing.T) {
	root := t.TempDir()

	// Create a file that imports a nonexistent internal package.
	if err := os.MkdirAll(filepath.Join(root, "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `package main

import "gitlab.com/tinyland/lab/prompt-pulse/nonexistent/package"

func main() {}
`
	if err := os.WriteFile(filepath.Join(root, "main", "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	orphans, err := clFindOrphanedImports(root)
	if err != nil {
		t.Fatalf("find orphaned failed: %v", err)
	}
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if !strings.Contains(orphans[0], "nonexistent/package") {
		t.Errorf("unexpected orphan: %s", orphans[0])
	}
}

func TestFindOrphanedImportsNone(t *testing.T) {
	root := t.TempDir()

	// Create a file with only stdlib imports.
	if err := os.MkdirAll(filepath.Join(root, "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `package main

import "fmt"

func main() { fmt.Println("hi") }
`
	if err := os.WriteFile(filepath.Join(root, "main", "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	orphans, err := clFindOrphanedImports(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected no orphans, got %d", len(orphans))
	}
}

// --- Metrics tests ---

func TestComputeMetrics(t *testing.T) {
	root := clCreateTestTree(t)
	m, err := clComputeMetrics(root)
	if err != nil {
		t.Fatalf("compute metrics failed: %v", err)
	}
	if m.V1Files == 0 {
		t.Error("expected non-zero v1 file count")
	}
	if m.V1Lines == 0 {
		t.Error("expected non-zero v1 line count")
	}
	if m.V2Files == 0 {
		t.Error("expected non-zero v2 file count")
	}
	if m.V2Lines == 0 {
		t.Error("expected non-zero v2 line count")
	}
	if m.V1Packages == 0 {
		t.Error("expected non-zero v1 package count")
	}
	if m.V2Packages == 0 {
		t.Error("expected non-zero v2 package count")
	}
}

func TestComputeMetricsV2TestCount(t *testing.T) {
	root := clCreateTestTree(t)
	m, err := clComputeMetrics(root)
	if err != nil {
		t.Fatalf("compute metrics failed: %v", err)
	}
	if m.V2TestCount == 0 {
		t.Error("expected non-zero v2 test count")
	}
}

func TestDuplicationRatio(t *testing.T) {
	v1 := []FileInfo{
		{Path: "a/layout.go", Package: "a"},
		{Path: "a/other.go", Package: "a"},
		{Path: "b/render.go", Package: "b"},
	}
	v2 := []FileInfo{
		{Path: "pkg/x/layout.go", Package: "x"},
	}
	ratio := clComputeDuplicationRatio(v1, v2)
	// Only layout.go matches by filename, so 1/3.
	if ratio < 0.3 || ratio > 0.4 {
		t.Errorf("expected ratio ~0.33, got %f", ratio)
	}
}

func TestDuplicationRatioZero(t *testing.T) {
	ratio := clComputeDuplicationRatio(nil, nil)
	if ratio != 0.0 {
		t.Errorf("expected 0.0 for empty inputs, got %f", ratio)
	}
}

func TestDuplicationRatioFull(t *testing.T) {
	v1 := []FileInfo{
		{Path: "a/layout.go", Package: "layout"},
	}
	v2 := []FileInfo{
		{Path: "pkg/layout/layout.go", Package: "layout"},
	}
	ratio := clComputeDuplicationRatio(v1, v2)
	if ratio != 1.0 {
		t.Errorf("expected 1.0 for full overlap, got %f", ratio)
	}
}

func TestEstimateDebt(t *testing.T) {
	m := &CodeMetrics{
		V1Lines:          5000,
		V1Files:          50,
		V1Packages:       10,
		V2TestCount:      100,
		DuplicationRatio: 0.75,
	}
	summary := clEstimateDebt(m)
	if !strings.Contains(summary, "5000 lines") {
		t.Errorf("expected line count in summary: %s", summary)
	}
	if !strings.Contains(summary, "75%") {
		t.Errorf("expected duplication percentage: %s", summary)
	}
}

func TestEstimateDebtNil(t *testing.T) {
	summary := clEstimateDebt(nil)
	if summary != "no metrics available" {
		t.Errorf("unexpected nil summary: %s", summary)
	}
}

func TestEstimateDebtClean(t *testing.T) {
	m := &CodeMetrics{}
	summary := clEstimateDebt(m)
	if !strings.Contains(summary, "clean") {
		t.Errorf("expected clean message: %s", summary)
	}
}

// --- Safety tests ---

func TestIsSafeDeletionKnown(t *testing.T) {
	safe, reason := clIsSafeDeletion("display/banner", []string{"pkg/layout", "pkg/terminal"})
	if !safe {
		t.Errorf("display/banner should be safe: %s", reason)
	}
	if !strings.Contains(reason, "replaced") {
		t.Errorf("reason should mention replacement: %s", reason)
	}
}

func TestIsSafeDeletionUnknown(t *testing.T) {
	safe, reason := clIsSafeDeletion("unknown/dir", nil)
	if safe {
		t.Error("unknown directory should not be safe")
	}
	if !strings.Contains(reason, "manual review") {
		t.Errorf("reason should suggest manual review: %s", reason)
	}
}

func TestIsSafeDeletionUnsafeDir(t *testing.T) {
	safe, reason := clIsSafeDeletion("scripts", nil)
	if safe {
		t.Error("scripts should not be safe")
	}
	if !strings.Contains(reason, "manual review") {
		t.Errorf("reason should suggest manual review: %s", reason)
	}
}

func TestIsSafeDeletionSubpath(t *testing.T) {
	safe, _ := clIsSafeDeletion("display/banner/layout.go", []string{"pkg/layout"})
	if !safe {
		t.Error("subpath of known safe directory should be safe")
	}
}

func TestCheckReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `package main

func main() {
	x := Layout()
	_ = x
}
`
	if err := os.WriteFile(filepath.Join(root, "src", "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	refs, err := clCheckReferences("Layout", root)
	if err != nil {
		t.Fatalf("check references failed: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}
}

func TestCheckReferencesEmpty(t *testing.T) {
	refs, err := clCheckReferences("", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 {
		t.Error("expected no references for empty symbol")
	}
}

func TestCheckTestCoverage(t *testing.T) {
	root := clCreateTestTree(t)
	covered := clCheckTestCoverage(filepath.Join(root, "pkg", "layout"))
	if !covered {
		t.Error("pkg/layout should have test coverage")
	}
}

func TestCheckTestCoverageNone(t *testing.T) {
	root := clCreateTestTree(t)
	covered := clCheckTestCoverage(filepath.Join(root, "pkg", "data"))
	if covered {
		t.Error("pkg/data should not have test coverage in mock tree")
	}
}

func TestCheckTestCoverageNonexistent(t *testing.T) {
	covered := clCheckTestCoverage("/nonexistent/path")
	if covered {
		t.Error("nonexistent path should not have coverage")
	}
}

func TestGenerateSafetyReport(t *testing.T) {
	m := &CleanupManifest{
		Deletions: []Deletion{
			{Path: "display/banner", Reason: "replaced", Safe: true, LinesRemoved: 100, ReplacedBy: "pkg/layout/"},
			{Path: "scripts", Reason: "no replacement", Safe: false, LinesRemoved: 10},
		},
		Modifications: []Modification{
			{Path: "main.go", OldImport: "old", NewImport: "new"},
		},
		Summary: ManifestSummary{
			TotalDeletions:     2,
			SafeDeletions:      1,
			UnsafeDeletions:    1,
			TotalLinesRemoved:  110,
			TotalModifications: 1,
		},
	}

	report := clGenerateSafetyReport(m)
	if !strings.Contains(report, "Safety Analysis") {
		t.Error("missing report header")
	}
	if !strings.Contains(report, "Unsafe") {
		t.Error("missing unsafe section")
	}
	if !strings.Contains(report, "Safe") {
		t.Error("missing safe section")
	}
	if !strings.Contains(report, "Import Modifications") {
		t.Error("missing modifications section")
	}
}

func TestGenerateSafetyReportNil(t *testing.T) {
	report := clGenerateSafetyReport(nil)
	if !strings.Contains(report, "No deletions") {
		t.Errorf("expected no-deletions message: %s", report)
	}
}

func TestGenerateSafetyReportEmpty(t *testing.T) {
	m := &CleanupManifest{}
	report := clGenerateSafetyReport(m)
	if !strings.Contains(report, "No deletions") {
		t.Errorf("expected no-deletions message for empty manifest: %s", report)
	}
}

// --- End-to-end tests ---

func TestAnalyzeFullPipeline(t *testing.T) {
	root := clCreateTestTree(t)
	manifest, err := Analyze(root)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	// Should have found deletions for v1 directories that exist in the mock tree.
	if len(manifest.Deletions) == 0 {
		t.Error("expected deletions in manifest")
	}

	// Summary should be computed.
	if manifest.Summary.TotalDeletions != len(manifest.Deletions) {
		t.Errorf("summary mismatch: %d vs %d", manifest.Summary.TotalDeletions, len(manifest.Deletions))
	}

	// All deletion paths should be non-empty.
	for _, d := range manifest.Deletions {
		if d.Path == "" {
			t.Error("deletion with empty path")
		}
		if d.Reason == "" {
			t.Error("deletion with empty reason")
		}
	}
}

func TestAnalyzeEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	manifest, err := Analyze(root)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if len(manifest.Deletions) != 0 {
		t.Errorf("expected 0 deletions for empty dir, got %d", len(manifest.Deletions))
	}
}

func TestAnalyzeMarkdownPipeline(t *testing.T) {
	root := clCreateTestTree(t)
	manifest, err := Analyze(root)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	md := manifest.RenderMarkdown()
	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
	if !strings.Contains(md, "Cleanup Manifest") {
		t.Error("missing manifest header in pipeline output")
	}
}

func TestAnalyzeScriptPipeline(t *testing.T) {
	root := clCreateTestTree(t)
	manifest, err := Analyze(root)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	script := manifest.RenderScript()
	if len(script) == 0 {
		t.Error("expected non-empty script")
	}
	if !strings.Contains(script, "bash") {
		t.Error("script should reference bash")
	}
}

// --- Edge case tests ---

func TestScanDirectoryWithBinaryFiles(t *testing.T) {
	root := t.TempDir()
	// Create a binary file alongside a Go file.
	if err := os.WriteFile(filepath.Join(root, "binary.exe"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	goContent := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(goContent), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := clScanDirectory(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 Go file (ignoring binary), got %d", len(files))
	}
}

func TestScanDirectoryWithSymlinks(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	goContent := "package sub\n\nfunc Hello() {}\n"
	if err := os.WriteFile(filepath.Join(sub, "hello.go"), []byte(goContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a symlink.
	link := filepath.Join(root, "link")
	if err := os.Symlink(sub, link); err != nil {
		t.Skip("symlinks not supported on this platform")
	}
	files, err := clScanDirectory(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	// Should find the file in sub and possibly through the symlink.
	if len(files) == 0 {
		t.Error("expected at least 1 file")
	}
}

func TestScanDirectoryNoGoFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "script.sh"), []byte("#!/bin/bash"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := clScanDirectory(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 Go files, got %d", len(files))
	}
}

func TestParseGoFileNonexistent(t *testing.T) {
	_, err := clParseGoFile("/nonexistent/file.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExtractQuotedImport(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{`"fmt"`, "fmt"},
		{`	"os/exec"`, "os/exec"},
		{`alias "gitlab.com/tinyland/lab/prompt-pulse/display/banner"`, "gitlab.com/tinyland/lab/prompt-pulse/display/banner"},
		{`no quotes here`, ""},
		{`"unclosed`, ""},
	}
	for _, tt := range tests {
		got := clExtractQuotedImport(tt.line)
		if got != tt.want {
			t.Errorf("extractQuotedImport(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestV1ImportMapCompleteness(t *testing.T) {
	m := clV1ImportMap()
	// Should have entries for all major v1 directories.
	expectedPrefixes := []string{
		"display/banner", "display/color", "display/layout", "display/render",
		"display/starship", "display/tui", "display/widgets",
		"waifu", "collectors", "shell", "config", "cache",
		"status", "internal/format",
	}
	const base = "gitlab.com/tinyland/lab/prompt-pulse/"
	for _, prefix := range expectedPrefixes {
		fullPath := base + prefix
		if _, ok := m[fullPath]; !ok {
			t.Errorf("missing import mapping for %s", fullPath)
		}
	}
}

func TestCountDirectoryLines(t *testing.T) {
	root := t.TempDir()
	f1 := "package a\n\nfunc A() {}\n"
	f2 := "package a\n\nfunc B() {}\n\nfunc C() {}\n"
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(f1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte(f2), 0o644); err != nil {
		t.Fatal(err)
	}

	total, err := clCountDirectoryLines(root)
	if err != nil {
		t.Fatalf("count dir lines failed: %v", err)
	}
	if total == 0 {
		t.Error("expected non-zero total lines")
	}
}

func TestMetricsWithEmptyRoot(t *testing.T) {
	root := t.TempDir()
	m, err := clComputeMetrics(root)
	if err != nil {
		t.Fatalf("compute metrics failed: %v", err)
	}
	if m.V1Files != 0 || m.V2Files != 0 {
		t.Errorf("expected zero files for empty root, got v1=%d v2=%d", m.V1Files, m.V2Files)
	}
}
