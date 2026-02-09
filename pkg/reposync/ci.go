package reposync

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// CIPipeline represents a complete GitLab CI pipeline definition.
type CIPipeline struct {
	// Stages lists the ordered pipeline stages.
	Stages []CIStage

	// Variables holds pipeline-level variables.
	Variables map[string]string

	// Rules holds pipeline-level rules.
	Rules []CIRule
}

// CIStage represents a single job within the pipeline.
type CIStage struct {
	// Name is the job name (also used as the stage name).
	Name string

	// Image is the Docker image for this job.
	Image string

	// Script holds the shell commands to execute.
	Script []string

	// Only restricts when this job runs (branch patterns).
	Only []string

	// Artifacts lists paths to preserve as job artifacts.
	Artifacts []string
}

// CIRule describes a conditional execution rule.
type CIRule struct {
	// If is a CI/CD variable expression.
	If string

	// When controls execution timing ("on_success", "manual", "always", etc.).
	When string

	// AllowFailure permits the job to fail without failing the pipeline.
	AllowFailure bool
}

// GenerateSyncPipeline builds a CIPipeline that synchronizes files from a
// monorepo source path to a standalone target repository.
func GenerateSyncPipeline(config *SyncConfig) (*CIPipeline, error) {
	if config == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if errs := ValidateConfig(config); len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %s", strings.Join(errs, "; "))
	}

	pipeline := &CIPipeline{
		Variables: map[string]string{
			"SOURCE_REPO":   config.SourceRepo,
			"SOURCE_PATH":   config.SourcePath,
			"TARGET_REPO":   config.TargetRepo,
			"TARGET_BRANCH": config.TargetBranch,
		},
	}

	stageNames := []string{"detect-changes", "prepare-sync", "validate-build", "push-sync"}
	for _, name := range stageNames {
		stage := CIStage{
			Name:   name,
			Image:  "golang:1.25",
			Script: rsBuildScript(name, config),
		}

		switch name {
		case "detect-changes":
			stage.Artifacts = []string{"changed_files.txt"}
		case "prepare-sync":
			stage.Artifacts = []string{"sync_workspace/"}
		case "push-sync":
			stage.Only = []string{"main"}
		}

		pipeline.Stages = append(pipeline.Stages, stage)
	}

	pipeline.Rules = []CIRule{
		{If: `$CI_PIPELINE_SOURCE == "push"`, When: "on_success"},
		{If: `$CI_PIPELINE_SOURCE == "schedule"`, When: "always"},
	}

	return pipeline, nil
}

// rsRenderGitLabCI renders a CIPipeline as a GitLab CI YAML string.
func rsRenderGitLabCI(pipeline *CIPipeline) (string, error) {
	if pipeline == nil {
		return "", fmt.Errorf("pipeline must not be nil")
	}

	const tpl = `# Auto-generated sync pipeline
# Do not edit manually

stages:
{{- range .Stages}}
  - {{.Name}}
{{- end}}

variables:
{{- range $k, $v := .Variables}}
  {{$k}}: "{{$v}}"
{{- end}}

{{- range .Stages}}

{{.Name}}:
  stage: {{.Name}}
  image: {{.Image}}
  script:
{{- range .Script}}
    - {{.}}
{{- end}}
{{- if .Only}}
  only:
{{- range .Only}}
    - {{.}}
{{- end}}
{{- end}}
{{- if .Artifacts}}
  artifacts:
    paths:
{{- range .Artifacts}}
      - {{.}}
{{- end}}
{{- end}}
{{- end}}

{{- if .Rules}}

workflow:
  rules:
{{- range .Rules}}
    - if: '{{.If}}'
      when: {{.When}}
{{- if .AllowFailure}}
      allow_failure: true
{{- end}}
{{- end}}
{{- end}}
`

	tmpl, err := template.New("gitlab-ci").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pipeline); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// rsBuildScript generates the shell commands for a given pipeline stage.
func rsBuildScript(stage string, config *SyncConfig) []string {
	switch stage {
	case "detect-changes":
		return []string{
			fmt.Sprintf("cd %s", config.SourcePath),
			"git diff --name-only HEAD~1 HEAD > changed_files.txt || true",
			fmt.Sprintf(`grep -E '^(%s)' changed_files.txt > sync_changes.txt || true`,
				strings.Join(rsEscapePaths(config.SyncPaths), "|")),
			`if [ ! -s sync_changes.txt ]; then echo "No sync-relevant changes"; exit 0; fi`,
			`echo "$(wc -l < sync_changes.txt) files changed"`,
		}

	case "prepare-sync":
		lines := []string{
			"mkdir -p sync_workspace",
		}
		for _, p := range config.SyncPaths {
			lines = append(lines,
				fmt.Sprintf("cp -r %s%s sync_workspace/ 2>/dev/null || true", config.SourcePath, p))
		}
		lines = append(lines,
			fmt.Sprintf(`sed -i 's|module .*|module %s|' sync_workspace/go.mod`,
				rsTargetModule(config.TargetRepo)))
		return lines

	case "validate-build":
		return []string{
			"cd sync_workspace",
			"go mod tidy",
			"go build ./...",
			"go test ./... -count=1 -short",
		}

	case "push-sync":
		return []string{
			fmt.Sprintf("git clone https://${SYNC_TOKEN}@%s target_repo", config.TargetRepo),
			fmt.Sprintf("cd target_repo && git checkout %s", config.TargetBranch),
			"rsync -av --delete sync_workspace/ target_repo/ --exclude .git",
			`cd target_repo && git add -A`,
			fmt.Sprintf(`cd target_repo && git commit -m "sync: update from %s@${CI_COMMIT_SHORT_SHA}" || true`, config.SourceRepo),
			"cd target_repo && git push origin HEAD",
		}

	default:
		return []string{fmt.Sprintf("echo 'Unknown stage: %s'", stage)}
	}
}

// rsEscapePaths converts sync paths to regex-safe patterns for grep.
func rsEscapePaths(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		p = strings.ReplaceAll(p, ".", `\.`)
		p = strings.ReplaceAll(p, "*", ".*")
		out[i] = p
	}
	return out
}

// rsTargetModule derives the Go module path from a repo URL.
func rsTargetModule(targetRepo string) string {
	return targetRepo
}
