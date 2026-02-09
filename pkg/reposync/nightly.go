package reposync

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// NightlyConfig describes a scheduled job that updates flake inputs and
// optionally creates merge requests.
type NightlyConfig struct {
	// Schedule is a cron expression (e.g., "0 3 * * *").
	Schedule string

	// FlakeInputs lists the flake input names to update.
	FlakeInputs []string

	// AutoMerge enables automatic merging of the update MR.
	AutoMerge bool

	// BranchPrefix is the git branch prefix for update branches.
	BranchPrefix string
}

// DefaultNightly returns the standard nightly update configuration.
func DefaultNightly() *NightlyConfig {
	return &NightlyConfig{
		Schedule:     "0 3 * * *",
		FlakeInputs:  []string{"prompt-pulse"},
		AutoMerge:    false,
		BranchPrefix: "nightly/flake-update",
	}
}

// rsGenerateNightlyJob renders a GitLab CI job YAML snippet for a nightly
// flake update schedule.
func rsGenerateNightlyJob(config *NightlyConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config must not be nil")
	}
	if err := rsValidateSchedule(config.Schedule); err != nil {
		return "", fmt.Errorf("invalid schedule: %w", err)
	}

	const tpl = `# Nightly flake input update job
nightly-flake-update:
  stage: maintenance
  image: nixos/nix:latest
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
      when: always
  variables:
    BRANCH_PREFIX: "{{ .BranchPrefix }}"
    AUTO_MERGE: "{{ .AutoMergeStr }}"
  script:
    - nix --version
{{- range .FlakeInputs }}
    - nix flake update {{ . }}
{{- end }}
    - |
      if git diff --quiet flake.lock; then
        echo "No changes to flake.lock"
        exit 0
      fi
    - git checkout -b "${BRANCH_PREFIX}/$(date +%Y%m%d)"
    - git add flake.lock
    - git commit -m "chore(nix): nightly flake input update"
    - git push origin HEAD
{{- if .AutoMerge }}
    - |
      curl --request POST \
        --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" \
        "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/merge_requests" \
        --form "source_branch=${BRANCH_PREFIX}/$(date +%Y%m%d)" \
        --form "target_branch=main" \
        --form "title=chore(nix): nightly flake update $(date +%Y-%m-%d)" \
        --form "merge_when_pipeline_succeeds=true"
{{- end }}
`

	autoMergeStr := "false"
	if config.AutoMerge {
		autoMergeStr = "true"
	}

	data := struct {
		BranchPrefix string
		AutoMergeStr string
		FlakeInputs  []string
		AutoMerge    bool
	}{
		BranchPrefix: config.BranchPrefix,
		AutoMergeStr: autoMergeStr,
		FlakeInputs:  config.FlakeInputs,
		AutoMerge:    config.AutoMerge,
	}

	tmpl, err := template.New("nightly").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// rsValidateSchedule checks that a cron expression has the expected number
// of fields and uses valid characters.
func rsValidateSchedule(cron string) error {
	cron = strings.TrimSpace(cron)
	if cron == "" {
		return fmt.Errorf("schedule must not be empty")
	}

	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return fmt.Errorf("expected 5 fields in cron expression, got %d", len(fields))
	}

	// Validate each field against allowed cron characters.
	cronFieldRe := regexp.MustCompile(`^[\d*,/\-]+$`)
	fieldNames := []string{"minute", "hour", "day-of-month", "month", "day-of-week"}
	for i, field := range fields {
		if !cronFieldRe.MatchString(field) {
			return fmt.Errorf("invalid %s field: %q", fieldNames[i], field)
		}
	}

	return nil
}
