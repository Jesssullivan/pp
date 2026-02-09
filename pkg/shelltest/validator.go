package shelltest

import (
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/shell"
)

// ValidationResult holds the result of validating a shell script.
type ValidationResult struct {
	Shell    string
	Valid    bool
	Errors   []string
	Warnings []string
}

// Validate checks a shell integration script for correctness. It performs
// structural validation (not execution) of the generated script by running
// pattern checks, syntax analysis, and security auditing.
func Validate(shellType shell.ShellType, script string) ValidationResult {
	result := ValidationResult{
		Shell: string(shellType),
		Valid: true,
	}

	// Pattern validation: check required/forbidden patterns.
	stValidatePatterns(shellType, script, &result)

	// Syntax validation: structural checks.
	stValidateSyntax(shellType, script, &result)

	// Security validation.
	stValidateSecurity(shellType, script, &result)

	return result
}

// ValidateAll generates and validates scripts for all shell types using
// the given options. It enables all features to maximize pattern coverage.
func ValidateAll(opts shell.Options) map[shell.ShellType]ValidationResult {
	shells := []shell.ShellType{shell.Bash, shell.Zsh, shell.Fish, shell.Ksh}
	results := make(map[shell.ShellType]ValidationResult, len(shells))

	for _, sh := range shells {
		script := shell.Generate(sh, opts)
		results[sh] = Validate(sh, script)
	}

	return results
}

// stValidatePatterns checks the script against the required and forbidden
// patterns for the given shell type.
func stValidatePatterns(shellType shell.ShellType, script string, result *ValidationResult) {
	patterns := PatternsFor(shellType)
	for _, p := range patterns {
		matched := p.Matches(script)
		if p.Required && !matched {
			result.Errors = append(result.Errors, "missing required pattern: "+p.Name+" - "+p.Description)
			result.Valid = false
		}
		if !p.Required && matched {
			result.Warnings = append(result.Warnings, "forbidden pattern found: "+p.Name+" - "+p.Description)
		}
	}
}

// stValidateSyntax runs the appropriate syntax checker for the shell type.
func stValidateSyntax(shellType shell.ShellType, script string, result *ValidationResult) {
	var syntaxErrors []string

	switch shellType {
	case shell.Bash:
		syntaxErrors = stValidateBashSyntax(script)
	case shell.Zsh:
		syntaxErrors = stValidateZshSyntax(script)
	case shell.Fish:
		syntaxErrors = stValidateFishSyntax(script)
	case shell.Ksh:
		syntaxErrors = stValidateKshSyntax(script)
	}

	if len(syntaxErrors) > 0 {
		result.Errors = append(result.Errors, syntaxErrors...)
		result.Valid = false
	}
}

// stValidateSecurity runs the security checks on the script.
func stValidateSecurity(shellType shell.ShellType, script string, result *ValidationResult) {
	// Injection checks.
	if warnings := stCheckInjection(script); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Permission checks.
	if warnings := stCheckPermissions(script); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Credential leak checks.
	if warnings := stCheckEnvLeaks(script); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}
}
