// Package docs generates documentation for prompt-pulse from embedded knowledge.
// It produces architecture docs, configuration references, shell integration guides,
// man pages, and changelogs in Markdown and roff formats.
package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Section represents a single documentation section with optional subsections.
type Section struct {
	// Title is the human-readable section heading.
	Title string

	// Slug is the URL/filename-safe identifier.
	Slug string

	// Content is the Markdown body of the section.
	Content string

	// Order controls the sort position (lower = earlier).
	Order int

	// SubSections are nested sections within this section.
	SubSections []Section
}

// DocGenerator orchestrates documentation generation across multiple sections.
type DocGenerator struct {
	// OutputDir is the directory where generated files are written.
	OutputDir string

	// Format is the output format ("markdown" or "roff"). Defaults to "markdown".
	Format string

	// Sections holds all top-level documentation sections.
	Sections []Section
}

// New creates a new DocGenerator that writes to outputDir.
func New(outputDir string) *DocGenerator {
	return &DocGenerator{
		OutputDir: outputDir,
		Format:    "markdown",
		Sections:  nil,
	}
}

// Add appends a new top-level section to the generator.
func (g *DocGenerator) Add(title, slug, content string, order int) {
	g.Sections = append(g.Sections, Section{
		Title:   title,
		Slug:    slug,
		Content: content,
		Order:   order,
	})
}

// AddSection appends a pre-built Section to the generator.
func (g *DocGenerator) AddSection(s Section) {
	g.Sections = append(g.Sections, s)
}

// Generate writes each section to its own file in OutputDir.
// Files are named "<slug>.md" for Markdown format.
func (g *DocGenerator) Generate() error {
	if err := os.MkdirAll(g.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	sorted := dcSortedSections(g.Sections)

	for _, s := range sorted {
		ext := ".md"
		if g.Format == "roff" {
			ext = ".1"
		}
		filename := filepath.Join(g.OutputDir, s.Slug+ext)
		body := dcRenderSection(s, 1)
		if err := os.WriteFile(filename, []byte(body), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}
	return nil
}

// GenerateSingle combines all sections into a single document string.
func (g *DocGenerator) GenerateSingle() (string, error) {
	sorted := dcSortedSections(g.Sections)

	var b strings.Builder
	b.WriteString("# prompt-pulse Documentation\n\n")

	// Table of contents
	b.WriteString("## Table of Contents\n\n")
	for i, s := range sorted {
		b.WriteString(fmt.Sprintf("%d. [%s](#%s)\n", i+1, s.Title, s.Slug))
	}
	b.WriteString("\n---\n\n")

	for _, s := range sorted {
		b.WriteString(dcRenderSection(s, 2))
		b.WriteString("\n---\n\n")
	}
	return b.String(), nil
}

// dcSortedSections returns a copy of sections sorted by Order.
func dcSortedSections(sections []Section) []Section {
	sorted := make([]Section, len(sections))
	copy(sorted, sections)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})
	return sorted
}

// dcRenderSection renders a section and its subsections as Markdown.
func dcRenderSection(s Section, level int) string {
	var b strings.Builder
	prefix := strings.Repeat("#", level)
	b.WriteString(fmt.Sprintf("%s %s\n\n", prefix, s.Title))

	if s.Content != "" {
		b.WriteString(s.Content)
		if !strings.HasSuffix(s.Content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(s.SubSections) > 0 {
		sub := dcSortedSections(s.SubSections)
		for _, ss := range sub {
			b.WriteString(dcRenderSection(ss, level+1))
		}
	}
	return b.String()
}
