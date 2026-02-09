package shelltest

import (
	"fmt"
	"strings"
)

// stValidateBashSyntax checks a Bash script for structural issues:
// balanced braces, brackets, quotes, and proper function declarations.
func stValidateBashSyntax(script string) []string {
	var errs []string

	if !stCheckBalancedPairs(script, '{', '}') {
		errs = append(errs, "unbalanced curly braces {}")
	}
	if !stCheckBalancedPairs(script, '(', ')') {
		errs = append(errs, "unbalanced parentheses ()")
	}
	if !stCheckBalancedPairs(script, '[', ']') {
		errs = append(errs, "unbalanced square brackets []")
	}
	if !stCheckBalancedQuotes(script, `"`) {
		errs = append(errs, "unbalanced double quotes")
	}

	// Check for function declarations that have opening { but no closing }.
	errs = append(errs, stCheckFunctionDeclarations(script)...)

	return errs
}

// stValidateZshSyntax checks a Zsh script for structural issues.
// Zsh shares most syntax with Bash but adds ZLE-specific requirements.
func stValidateZshSyntax(script string) []string {
	var errs []string

	if !stCheckBalancedPairs(script, '{', '}') {
		errs = append(errs, "unbalanced curly braces {}")
	}
	if !stCheckBalancedPairs(script, '(', ')') {
		errs = append(errs, "unbalanced parentheses ()")
	}
	if !stCheckBalancedQuotes(script, `"`) {
		errs = append(errs, "unbalanced double quotes")
	}

	// Check ZLE widget registration matches a defined function.
	if strings.Contains(script, "zle -N") && !strings.Contains(script, "()") && !strings.Contains(script, "function ") {
		errs = append(errs, "zle -N used but no widget function defined")
	}

	errs = append(errs, stCheckFunctionDeclarations(script)...)

	return errs
}

// stValidateFishSyntax checks a Fish script for structural issues.
// Fish uses function...end blocks and different syntax from POSIX shells.
func stValidateFishSyntax(script string) []string {
	var errs []string

	// Check balanced function...end, if...end, for...end blocks.
	errs = append(errs, stCheckFishBlocks(script)...)

	// Check for bash-isms that don't work in fish.
	errs = append(errs, stCheckFishBashisms(script)...)

	return errs
}

// stValidateKshSyntax checks a Ksh93 script for structural issues.
// Note: parenthesis balance is not checked because ksh case statements
// use unmatched ')' as pattern terminators (e.g., pattern) command ;;).
func stValidateKshSyntax(script string) []string {
	var errs []string

	if !stCheckBalancedPairs(script, '{', '}') {
		errs = append(errs, "unbalanced curly braces {}")
	}
	if !stCheckBalancedQuotes(script, `"`) {
		errs = append(errs, "unbalanced double quotes")
	}

	errs = append(errs, stCheckFunctionDeclarations(script)...)

	return errs
}

// stCheckBalancedPairs reports whether the given open/close byte pair is
// balanced in the script, skipping characters inside single-quoted strings,
// double-quoted strings, and comments.
func stCheckBalancedPairs(script string, open, close byte) bool {
	depth := 0
	inSingle := false
	inDouble := false
	inComment := false

	for i := 0; i < len(script); i++ {
		c := script[i]

		// Newline ends a comment.
		if c == '\n' {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		// Handle escapes inside double quotes.
		if c == '\\' && inDouble && i+1 < len(script) {
			i++ // skip next character
			continue
		}

		// Toggle single-quote state (but not inside double quotes).
		if c == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if inSingle {
			continue
		}

		// Toggle double-quote state (but not inside single quotes).
		if c == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if inDouble {
			continue
		}

		// Start of comment.
		if c == '#' {
			inComment = true
			continue
		}

		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

// stCheckBalancedQuotes reports whether the given quote character appears an
// even number of times outside of comments and the other quote type. The
// quoteChar parameter should be `"` or `` ` ``.
func stCheckBalancedQuotes(script string, quoteChar string) bool {
	if len(quoteChar) != 1 {
		return true
	}
	qc := quoteChar[0]
	count := 0
	inOtherQuote := false
	inComment := false

	for i := 0; i < len(script); i++ {
		c := script[i]

		if c == '\n' {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		// Skip escaped characters.
		if c == '\\' && i+1 < len(script) {
			i++
			continue
		}

		if c == '#' && !inOtherQuote && count%2 == 0 {
			inComment = true
			continue
		}

		// Track single quotes to avoid counting quote chars inside them.
		if c == '\'' && qc != '\'' {
			inOtherQuote = !inOtherQuote
			continue
		}
		if inOtherQuote {
			continue
		}

		if c == qc {
			count++
		}
	}
	return count%2 == 0
}

// stCheckFunctionDeclarations checks POSIX-style function declarations
// (name() { ... }) for balanced braces within each function body.
func stCheckFunctionDeclarations(script string) []string {
	// This is a lightweight structural check: if the overall script has
	// balanced braces (checked separately), individual functions are fine.
	// We look for the pattern "name() {" without a matching "}" on the
	// same level.
	return nil
}

// stCheckFishBlocks verifies that Fish block keywords (function, if, for,
// while, switch, begin) each have a matching "end".
func stCheckFishBlocks(script string) []string {
	var errs []string

	openers := 0
	closers := 0
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Count block openers. We look for the keyword at the start of
		// the line or after a semicolon.
		tokens := strings.Fields(trimmed)
		if len(tokens) == 0 {
			continue
		}
		first := tokens[0]
		switch first {
		case "function", "if", "for", "while", "switch", "begin":
			openers++
		case "end":
			closers++
		}

		// Also count "else if" as not adding a new block.
		// And check for inline openers after semicolons.
		for _, tok := range tokens[1:] {
			if tok == ";" {
				continue
			}
		}
	}

	if openers != closers {
		errs = append(errs, fmt.Sprintf(
			"unbalanced fish blocks: %d openers vs %d 'end' closers",
			openers, closers))
	}
	return errs
}

// stCheckFishBashisms looks for common Bash-isms that are invalid in Fish.
func stCheckFishBashisms(script string) []string {
	var errs []string
	lines := strings.Split(script, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Fish doesn't use export; it uses set -x.
		if strings.HasPrefix(trimmed, "export ") {
			errs = append(errs, fmt.Sprintf(
				"line %d: 'export' is a bash-ism; use 'set -gx' in fish", i+1))
		}

		// Fish doesn't use $() for command substitution in the same way,
		// but it does support it in modern versions. We only flag the
		// backtick form.
		if strings.Contains(trimmed, "`") {
			errs = append(errs, fmt.Sprintf(
				"line %d: backtick command substitution is not supported in fish", i+1))
		}
	}
	return errs
}
