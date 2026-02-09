package reposync

import (
	"fmt"
	"regexp"
	"strings"
)

// rsRewriteGoMod replaces the module declaration in go.mod content, changing
// oldModule to newModule. It also rewrites any replace directives that
// reference the old module path.
func rsRewriteGoMod(content string, oldModule, newModule string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("go.mod content is empty")
	}
	if strings.TrimSpace(oldModule) == "" {
		return "", fmt.Errorf("old module path must not be empty")
	}
	if strings.TrimSpace(newModule) == "" {
		return "", fmt.Errorf("new module path must not be empty")
	}
	if oldModule == newModule {
		return content, nil
	}

	// Rewrite the module line.
	moduleRe := regexp.MustCompile(`(?m)^module\s+` + regexp.QuoteMeta(oldModule) + `\s*$`)
	if !moduleRe.MatchString(content) {
		return "", fmt.Errorf("module declaration %q not found in go.mod", oldModule)
	}
	result := moduleRe.ReplaceAllString(content, "module "+newModule)

	// Rewrite any replace directives referencing the old module.
	result = strings.ReplaceAll(result, oldModule+"/", newModule+"/")

	return result, nil
}

// rsRewriteImports replaces all import paths in a Go source file, changing
// oldModule prefixed imports to use newModule.
func rsRewriteImports(goFileContent string, oldModule, newModule string) (string, error) {
	if strings.TrimSpace(oldModule) == "" {
		return "", fmt.Errorf("old module path must not be empty")
	}
	if strings.TrimSpace(newModule) == "" {
		return "", fmt.Errorf("new module path must not be empty")
	}
	if oldModule == newModule {
		return goFileContent, nil
	}

	// Replace all occurrences of the old module path in import statements.
	// This handles both quoted imports in import blocks and single imports.
	result := strings.ReplaceAll(goFileContent, `"`+oldModule, `"`+newModule)

	return result, nil
}

// rsDetectModule extracts the module path from go.mod content.
func rsDetectModule(goModContent string) (string, error) {
	if strings.TrimSpace(goModContent) == "" {
		return "", fmt.Errorf("go.mod content is empty")
	}

	moduleRe := regexp.MustCompile(`(?m)^module\s+(\S+)\s*$`)
	matches := moduleRe.FindStringSubmatch(goModContent)
	if len(matches) < 2 {
		return "", fmt.Errorf("no module declaration found in go.mod")
	}

	return matches[1], nil
}

// rsValidateGoMod checks go.mod content for common issues and returns a list
// of warning/error strings. An empty return means no issues detected.
func rsValidateGoMod(content string) []string {
	var issues []string

	if strings.TrimSpace(content) == "" {
		return []string{"go.mod is empty"}
	}

	// Check for module declaration.
	moduleRe := regexp.MustCompile(`(?m)^module\s+\S+`)
	if !moduleRe.MatchString(content) {
		issues = append(issues, "missing module declaration")
	}

	// Check for go directive.
	goRe := regexp.MustCompile(`(?m)^go\s+\d+\.\d+`)
	if !goRe.MatchString(content) {
		issues = append(issues, "missing go version directive")
	}

	// Check for replace directives (common in monorepos, problematic in standalone).
	replaceRe := regexp.MustCompile(`(?m)^replace\s+`)
	if replaceRe.MatchString(content) {
		issues = append(issues, "contains replace directives (may need adjustment for standalone repo)")
	}

	// Check for local path replacements which won't work in standalone.
	localReplaceRe := regexp.MustCompile(`(?m)^replace\s+\S+\s+=>\s+\./`)
	if localReplaceRe.MatchString(content) {
		issues = append(issues, "contains local path replace directives (will break in standalone repo)")
	}

	return issues
}
