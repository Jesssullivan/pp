package shelltest

import (
	"fmt"
	"regexp"
	"strings"
)

// Pre-compiled regexes for security checks.
var (
	stBareEvalRe   = regexp.MustCompile(`\beval\s+\$\(`)
	stChmod777Re   = regexp.MustCompile(`chmod\s+777`)
	stChmodWorldRe = regexp.MustCompile(`chmod\s+[0-7]?[0-7][67][67]`)
	stCredLeakRe   = regexp.MustCompile(`(?i)(password|secret|api_key|token)\s*=\s*['\"][^'\"]+['\"]`)
	stCurlAuthRe   = regexp.MustCompile(`curl\b.*(-u|--user)\s+\S+:\S+`)
)

// stCheckInjection checks for potential command injection patterns in
// a shell script. Returns a list of warnings.
func stCheckInjection(script string) []string {
	var warnings []string

	// Check for eval $( without quoting -- injection risk.
	if stBareEvalRe.MatchString(script) {
		warnings = append(warnings, "potential injection: 'eval $(' found without quoting; use 'eval \"$(...)\"' instead")
	}

	// Check for backtick command substitution (harder to nest safely).
	lines := strings.Split(script, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if stCountBackticks(trimmed)%2 != 0 {
			warnings = append(warnings, fmt.Sprintf(
				"line %d: unbalanced backtick; prefer $() for command substitution", i+1))
		}
	}

	return warnings
}

// stCountBackticks counts backtick characters outside of single and double
// quotes.
func stCountBackticks(line string) int {
	count := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '\\' && inDouble && i+1 < len(line) {
			i++
			continue
		}
		if c == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if c == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if c == '`' && !inSingle && !inDouble {
			count++
		}
	}
	return count
}

// stCheckQuoting verifies that binaryPath is properly quoted everywhere
// it appears in the script. A properly quoted path is wrapped in single
// quotes following POSIX conventions.
func stCheckQuoting(script, binaryPath string) []string {
	var warnings []string

	if binaryPath == "" {
		return nil
	}

	// If the binary path contains characters that require quoting,
	// verify every occurrence is inside single quotes.
	needsQuoting := strings.ContainsAny(binaryPath, " \t$\"'\\!;|&<>(){}[]?*~")
	if !needsQuoting {
		return nil
	}

	// Check that the unquoted form does not appear outside of comments.
	lines := strings.Split(script, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Look for the bare binary path not wrapped in quotes.
		if strings.Contains(line, binaryPath) {
			// Check if it is inside single quotes by looking for the
			// quoted form.
			quotedForm := "'" + binaryPath + "'"
			if !strings.Contains(line, quotedForm) {
				// Also check the escaped single-quote form.
				escapedForm := strings.ReplaceAll(binaryPath, "'", `'\''`)
				fullQuoted := "'" + escapedForm + "'"
				if !strings.Contains(line, fullQuoted) {
					warnings = append(warnings, fmt.Sprintf(
						"line %d: binary path %q appears unquoted", i+1, binaryPath))
				}
			}
		}
	}

	return warnings
}

// stCheckPermissions checks for overly permissive patterns like chmod 777,
// world-writable permissions, or unsafe umask values.
func stCheckPermissions(script string) []string {
	var warnings []string

	if stChmod777Re.MatchString(script) {
		warnings = append(warnings, "chmod 777 grants world read/write/execute permissions")
	}

	if stChmodWorldRe.MatchString(script) {
		// More specific check beyond just 777.
		matches := stChmodWorldRe.FindAllString(script, -1)
		for _, m := range matches {
			if !strings.Contains(m, "777") { // avoid duplicate with above
				warnings = append(warnings, fmt.Sprintf(
					"%q may grant overly permissive world access", m))
			}
		}
	}

	// Check for umask 000 or similarly permissive values.
	if strings.Contains(script, "umask 000") || strings.Contains(script, "umask 0000") {
		warnings = append(warnings, "umask 000 allows world read/write on new files")
	}

	return warnings
}

// stCheckEnvLeaks checks for accidental credential exposure patterns,
// such as hardcoded passwords, API keys, or tokens in the script.
func stCheckEnvLeaks(script string) []string {
	var warnings []string

	if stCredLeakRe.MatchString(script) {
		matches := stCredLeakRe.FindAllString(script, -1)
		for _, m := range matches {
			warnings = append(warnings, fmt.Sprintf(
				"potential credential leak: %q", m))
		}
	}

	if stCurlAuthRe.MatchString(script) {
		warnings = append(warnings, "curl with inline credentials detected; use -n or .netrc instead")
	}

	return warnings
}
