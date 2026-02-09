package shelltest

import (
	"strings"
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/shell"
)

// stFullOpts returns Options with all features enabled, which is the
// recommended production configuration for testing all patterns.
func stFullOpts() shell.Options {
	return shell.Options{
		BinaryPath:        "/usr/local/bin/prompt-pulse",
		ShowBanner:        true,
		DaemonAutoStart:   true,
		EnableCompletions: true,
	}
}

// --- Validate tests for each shell type ---

func TestValidate_BashDefaultOptionsPasses(t *testing.T) {
	script := shell.Generate(shell.Bash, stFullOpts())
	result := Validate(shell.Bash, script)
	if !result.Valid {
		t.Errorf("Bash validation failed: %v", result.Errors)
	}
}

func TestValidate_ZshDefaultOptionsPasses(t *testing.T) {
	script := shell.Generate(shell.Zsh, stFullOpts())
	result := Validate(shell.Zsh, script)
	if !result.Valid {
		t.Errorf("Zsh validation failed: %v", result.Errors)
	}
}

func TestValidate_FishDefaultOptionsPasses(t *testing.T) {
	script := shell.Generate(shell.Fish, stFullOpts())
	result := Validate(shell.Fish, script)
	if !result.Valid {
		t.Errorf("Fish validation failed: %v", result.Errors)
	}
}

func TestValidate_KshDefaultOptionsPasses(t *testing.T) {
	script := shell.Generate(shell.Ksh, stFullOpts())
	result := Validate(shell.Ksh, script)
	if !result.Valid {
		t.Errorf("Ksh validation failed: %v", result.Errors)
	}
}

// --- ValidateAll tests ---

func TestValidateAll_ReturnsResultsForAllShells(t *testing.T) {
	results := ValidateAll(stFullOpts())
	expected := []shell.ShellType{shell.Bash, shell.Zsh, shell.Fish, shell.Ksh}
	for _, sh := range expected {
		if _, ok := results[sh]; !ok {
			t.Errorf("ValidateAll missing result for shell %s", sh)
		}
	}
	if len(results) != len(expected) {
		t.Errorf("ValidateAll returned %d results, want %d", len(results), len(expected))
	}
}

func TestValidateAll_AllShellsPassWithDefaultOptions(t *testing.T) {
	results := ValidateAll(stFullOpts())
	for sh, result := range results {
		if !result.Valid {
			t.Errorf("shell %s failed validation: %v", sh, result.Errors)
		}
	}
}

// --- PatternsFor tests ---

func TestPatternsFor_BashReturnsExpectedCount(t *testing.T) {
	patterns := PatternsFor(shell.Bash)
	// At minimum we expect: PROMPT_COMMAND, bind, complete -C,
	// pp-start, pp-stop, pp-status, pp-banner, quoted_binary,
	// bare_eval_injection, unbalanced_braces = 10
	if len(patterns) < 8 {
		t.Errorf("PatternsFor(Bash) returned %d patterns, want >= 8", len(patterns))
	}
}

func TestPatternsFor_EachShellHasPpStartPattern(t *testing.T) {
	shells := []shell.ShellType{shell.Bash, shell.Zsh, shell.Fish, shell.Ksh}
	for _, sh := range shells {
		patterns := PatternsFor(sh)
		found := false
		for _, p := range patterns {
			if p.Name == "pp_start" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PatternsFor(%s) missing pp_start pattern", sh)
		}
	}
}

// --- Bash-specific script content tests ---

func TestBashScript_ContainsPromptCommand(t *testing.T) {
	script := shell.Generate(shell.Bash, stFullOpts())
	if !strings.Contains(script, "PROMPT_COMMAND") {
		t.Error("Bash script with ShowBanner should contain PROMPT_COMMAND")
	}
}

func TestBashScript_ContainsBind(t *testing.T) {
	script := shell.Generate(shell.Bash, stFullOpts())
	if !strings.Contains(script, "bind") {
		t.Error("Bash script should contain bind for keybinding")
	}
}

// --- Zsh-specific script content tests ---

func TestZshScript_ContainsAddZshHook(t *testing.T) {
	script := shell.Generate(shell.Zsh, stFullOpts())
	if !strings.Contains(script, "add-zsh-hook") {
		t.Error("Zsh script with ShowBanner should contain add-zsh-hook")
	}
}

func TestZshScript_ContainsDevTty(t *testing.T) {
	script := shell.Generate(shell.Zsh, stFullOpts())
	if !strings.Contains(script, "/dev/tty") {
		t.Error("Zsh script should contain /dev/tty for TUI widget")
	}
}

// --- Fish-specific script content tests ---

func TestFishScript_ContainsAllThreeBindModes(t *testing.T) {
	script := shell.Generate(shell.Fish, stFullOpts())
	if !strings.Contains(script, "-M insert") {
		t.Error("Fish script should bind in insert mode")
	}
	if !strings.Contains(script, "-M visual") {
		t.Error("Fish script should bind in visual mode")
	}
	// Default mode is the bare bind without -M flag.
	lines := strings.Split(script, "\n")
	foundDefaultBind := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "bind ") && !strings.Contains(trimmed, "-M ") {
			foundDefaultBind = true
			break
		}
	}
	if !foundDefaultBind {
		t.Error("Fish script should have a bare bind for default mode")
	}
}

func TestFishScript_ContainsOnEventFishPrompt(t *testing.T) {
	script := shell.Generate(shell.Fish, stFullOpts())
	if !strings.Contains(script, "--on-event fish_prompt") {
		t.Error("Fish script with ShowBanner should contain --on-event fish_prompt")
	}
}

// --- Ksh-specific script content tests ---

func TestKshScript_ContainsTrapKeybd(t *testing.T) {
	script := shell.Generate(shell.Ksh, stFullOpts())
	if !strings.Contains(script, "trap") {
		t.Error("Ksh script should contain trap")
	}
	if !strings.Contains(script, "KEYBD") {
		t.Error("Ksh script should contain KEYBD")
	}
}

// --- Syntax validation tests ---

func TestStCheckBalancedPairs_Balanced(t *testing.T) {
	script := `func() { echo "hello {world}"; }`
	if !stCheckBalancedPairs(script, '{', '}') {
		t.Error("balanced braces should return true")
	}
}

func TestStCheckBalancedPairs_Unbalanced(t *testing.T) {
	script := `func() { echo "hello"; `
	if stCheckBalancedPairs(script, '{', '}') {
		t.Error("unbalanced braces should return false")
	}
}

func TestStCheckBalancedQuotes_Balanced(t *testing.T) {
	script := `echo "hello" && echo "world"`
	if !stCheckBalancedQuotes(script, `"`) {
		t.Error("balanced double quotes should return true")
	}
}

func TestStCheckBalancedQuotes_Unbalanced(t *testing.T) {
	script := `echo "hello`
	if stCheckBalancedQuotes(script, `"`) {
		t.Error("unbalanced double quotes should return false")
	}
}

// --- Security validation tests ---

func TestStCheckInjection_DetectsBareEval(t *testing.T) {
	script := `eval $(some_command)`
	warnings := stCheckInjection(script)
	if len(warnings) == 0 {
		t.Error("stCheckInjection should detect bare eval $()")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "injection") || strings.Contains(w, "eval") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("warning should mention injection/eval, got: %v", warnings)
	}
}

func TestStCheckInjection_AllowsQuotedEval(t *testing.T) {
	script := `eval "$(some_command)"`
	warnings := stCheckInjection(script)
	for _, w := range warnings {
		if strings.Contains(w, "injection") {
			t.Errorf("quoted eval should not trigger injection warning, got: %s", w)
		}
	}
}

func TestStCheckQuoting_VerifiesBinaryPathQuoted(t *testing.T) {
	binaryPath := "/path with spaces/prompt-pulse"
	script := `'/path with spaces/prompt-pulse' banner`
	warnings := stCheckQuoting(script, binaryPath)
	if len(warnings) != 0 {
		t.Errorf("properly quoted binary path should not produce warnings, got: %v", warnings)
	}
}

func TestStCheckQuoting_DetectsUnquotedBinaryPath(t *testing.T) {
	binaryPath := "/path with spaces/prompt-pulse"
	script := `/path with spaces/prompt-pulse banner`
	warnings := stCheckQuoting(script, binaryPath)
	if len(warnings) == 0 {
		t.Error("unquoted binary path with spaces should produce a warning")
	}
}

// --- Version compatibility tests ---

func TestCheckVersionCompat_Bash44Passes(t *testing.T) {
	script := shell.Generate(shell.Bash, stFullOpts())
	warnings := CheckVersionCompat(shell.Bash, "4.4", script)
	// Bash 4.4+ should have no PS0 or bind-x warnings for our generated script.
	for _, w := range warnings {
		if strings.Contains(w, "PS0") {
			t.Errorf("Bash 4.4+ should not warn about PS0: %s", w)
		}
	}
}

func TestCheckVersionCompat_Bash43WarnsAboutPS0(t *testing.T) {
	// Create a script that uses PS0 (our generated scripts don't, but
	// this tests the checker itself).
	script := `PS0='starting command...'`
	warnings := CheckVersionCompat(shell.Bash, "4.3", script)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "PS0") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Bash 4.3 should warn about PS0 usage")
	}
}

func TestCheckVersionCompat_Ksh88WarnsAboutKeybd(t *testing.T) {
	script := shell.Generate(shell.Ksh, stFullOpts())
	warnings := CheckVersionCompat(shell.Ksh, "88", script)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "KEYBD") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Ksh 88 should warn about KEYBD trap usage")
	}
}

// --- KnownVersions test ---

func TestKnownVersions_ReturnsExpectedShells(t *testing.T) {
	versions := KnownVersions()
	shellsSeen := make(map[shell.ShellType]bool)
	for _, v := range versions {
		shellsSeen[v.Shell] = true
	}
	expected := []shell.ShellType{shell.Bash, shell.Zsh, shell.Fish, shell.Ksh}
	for _, sh := range expected {
		if !shellsSeen[sh] {
			t.Errorf("KnownVersions missing shell: %s", sh)
		}
	}
}

// --- Security check passes for default scripts ---

func TestSecurityCheck_PassesForDefaultScripts(t *testing.T) {
	shells := []shell.ShellType{shell.Bash, shell.Zsh, shell.Fish, shell.Ksh}
	for _, sh := range shells {
		script := shell.Generate(sh, stFullOpts())
		injectionWarnings := stCheckInjection(script)
		for _, w := range injectionWarnings {
			if strings.Contains(w, "injection") {
				t.Errorf("default %s script triggered injection warning: %s", sh, w)
			}
		}
		permWarnings := stCheckPermissions(script)
		if len(permWarnings) > 0 {
			t.Errorf("default %s script triggered permission warnings: %v", sh, permWarnings)
		}
		leakWarnings := stCheckEnvLeaks(script)
		if len(leakWarnings) > 0 {
			t.Errorf("default %s script triggered env leak warnings: %v", sh, leakWarnings)
		}
	}
}

// --- Additional tests for full coverage ---

func TestStCheckBalancedPairs_CommentsIgnored(t *testing.T) {
	script := "# this has an { unbalanced brace in a comment\necho ok"
	if !stCheckBalancedPairs(script, '{', '}') {
		t.Error("braces in comments should be ignored")
	}
}

func TestStCheckBalancedPairs_QuotedIgnored(t *testing.T) {
	script := `echo "this has { unbalanced" && echo ok`
	if !stCheckBalancedPairs(script, '{', '}') {
		t.Error("braces inside double quotes should be ignored")
	}
}

func TestStCheckBalancedPairs_SingleQuotedIgnored(t *testing.T) {
	script := `echo 'this has { unbalanced' && echo ok`
	if !stCheckBalancedPairs(script, '{', '}') {
		t.Error("braces inside single quotes should be ignored")
	}
}

func TestValidate_UnknownShellReturnsValid(t *testing.T) {
	result := Validate(shell.ShellType("csh"), "# whatever")
	// Unknown shells have no patterns, so they pass.
	if !result.Valid {
		t.Errorf("unknown shell should pass validation (no patterns), got errors: %v", result.Errors)
	}
}

func TestFishSyntax_DetectsExportBashism(t *testing.T) {
	script := "export FOO=bar\n"
	errs := stValidateFishSyntax(script)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "export") {
			found = true
			break
		}
	}
	if !found {
		t.Error("fish syntax checker should flag 'export' as a bash-ism")
	}
}

func TestFishSyntax_DetectsUnbalancedBlocks(t *testing.T) {
	script := "function foo\n    echo hello\n"
	errs := stValidateFishSyntax(script)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "unbalanced") {
			found = true
			break
		}
	}
	if !found {
		t.Error("fish syntax checker should detect missing 'end' for function block")
	}
}

func TestStVersionLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"4.3", "4.4", true},
		{"4.4", "4.4", false},
		{"5.0", "4.4", false},
		{"3.2", "4.0", true},
		{"88", "93", true},
		{"93", "88", false},
	}
	for _, tt := range tests {
		got := stVersionLess(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("stVersionLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
