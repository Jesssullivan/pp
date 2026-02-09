package homebrew

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// ValidateFormula checks a generated Ruby formula string for common issues
// and returns a list of problems found. An empty slice means the formula
// passes all validation checks.
func ValidateFormula(ruby string) []string {
	var errs []string
	errs = append(errs, hbCheckRequiredSections(ruby)...)
	errs = append(errs, hbCheckRubyBalance(ruby)...)
	errs = append(errs, hbCheckClassName(ruby)...)
	errs = append(errs, hbCheckQuoteBalance(ruby)...)
	errs = append(errs, hbCheckTemplateArtifacts(ruby)...)
	return errs
}

// hbCheckRubyBalance verifies that do/end blocks and class/end blocks are
// balanced in the given Ruby source.
func hbCheckRubyBalance(ruby string) []string {
	var errs []string

	// Count block openers and end keywords
	lines := strings.Split(ruby, "\n")
	opens := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count block openers
		if strings.HasPrefix(trimmed, "class ") {
			opens++
		}
		if strings.HasPrefix(trimmed, "def ") {
			opens++
		}
		if strings.HasSuffix(trimmed, " do") || strings.HasSuffix(trimmed, " do\r") {
			opens++
		}
		// "service do" on its own line
		if trimmed == "do" {
			opens++
		}

		// Count end keywords (must be standalone or at line start)
		if trimmed == "end" {
			opens--
		}
	}

	if opens > 0 {
		errs = append(errs, fmt.Sprintf("unbalanced blocks: %d unclosed do/class/def", opens))
	} else if opens < 0 {
		errs = append(errs, fmt.Sprintf("unbalanced blocks: %d extra end keywords", -opens))
	}

	return errs
}

// hbCheckRequiredSections verifies that all required Homebrew formula sections
// are present in the Ruby source.
func hbCheckRequiredSections(ruby string) []string {
	var errs []string

	required := []struct {
		marker string
		label  string
	}{
		{`desc "`, "desc"},
		{`homepage "`, "homepage"},
		{`license "`, "license"},
		{"def install", "install"},
		{"test do", "test"},
	}

	for _, req := range required {
		if !strings.Contains(ruby, req.marker) {
			errs = append(errs, fmt.Sprintf("missing required section: %s", req.label))
		}
	}

	return errs
}

// hbCheckClassName verifies the class declaration follows the < Formula pattern.
func hbCheckClassName(ruby string) []string {
	var errs []string

	lines := strings.Split(ruby, "\n")
	foundClass := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "class ") {
			foundClass = true
			if !strings.Contains(trimmed, "< Formula") {
				errs = append(errs, "class must inherit from Formula")
			}
		}
	}

	if !foundClass {
		errs = append(errs, "missing class declaration")
	}

	return errs
}

// hbCheckQuoteBalance checks that double quotes are balanced on each line.
func hbCheckQuoteBalance(ruby string) []string {
	var errs []string

	lines := strings.Split(ruby, "\n")
	for i, line := range lines {
		count := 0
		escaped := false
		inString := false
		for _, r := range line {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' && inString {
				escaped = true
				continue
			}
			if r == '"' {
				count++
				inString = !inString
			}
		}
		if count%2 != 0 {
			errs = append(errs, fmt.Sprintf("unbalanced quotes on line %d", i+1))
		}
	}

	return errs
}

// hbCheckTemplateArtifacts checks for leftover Go template artifacts like {{.}}.
func hbCheckTemplateArtifacts(ruby string) []string {
	var errs []string
	if strings.Contains(ruby, "{{") || strings.Contains(ruby, "}}") {
		errs = append(errs, "template artifacts found ({{ or }}) in generated output")
	}
	if strings.Contains(ruby, "<no value>") {
		errs = append(errs, "template rendered <no value> placeholder")
	}
	return errs
}

// ValidatePlist checks that the given string is well-formed plist XML.
func ValidatePlist(xmlStr string) []string {
	var errs []string

	// Basic XML well-formedness check via decoder.
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			errs = append(errs, fmt.Sprintf("XML parse error: %v", err))
			break
		}
	}

	// Check for required plist elements.
	requiredElements := []string{
		"<plist",
		"<dict>",
		"<key>Label</key>",
		"<key>ProgramArguments</key>",
	}
	for _, elem := range requiredElements {
		if !strings.Contains(xmlStr, elem) {
			errs = append(errs, fmt.Sprintf("missing required plist element: %s", elem))
		}
	}

	return errs
}
