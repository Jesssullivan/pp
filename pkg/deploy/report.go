package deploy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DeployReport aggregates verification results across hosts.
type DeployReport struct {
	// Results holds per-host verification outcomes.
	Results []VerifyResult

	// Summary is the aggregated statistics.
	Summary ReportSummary

	// GeneratedAt is when the report was created.
	GeneratedAt time.Time
}

// ReportSummary holds aggregate counts across all hosts.
type ReportSummary struct {
	TotalHosts   int `json:"total_hosts"`
	PassedHosts  int `json:"passed_hosts"`
	FailedHosts  int `json:"failed_hosts"`
	TotalChecks  int `json:"total_checks"`
	PassedChecks int `json:"passed_checks"`
	FailedChecks int `json:"failed_checks"`
}

// NewReport creates a DeployReport from one or more VerifyResults and
// computes the summary.
func NewReport(results ...VerifyResult) *DeployReport {
	return &DeployReport{
		Results:     results,
		Summary:     dpComputeSummary(results),
		GeneratedAt: time.Now(),
	}
}

// RenderText returns a plain-text deployment report.
func (r *DeployReport) RenderText() string {
	var b strings.Builder

	b.WriteString("Deployment Verification Report\n")
	b.WriteString(strings.Repeat("=", 40))
	b.WriteString("\n\n")

	for _, res := range r.Results {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "Host: %s [%s]\n", res.Host, status)
		for _, cr := range res.Checks {
			mark := "+"
			if !cr.Passed {
				mark = "-"
			}
			fmt.Fprintf(&b, "  [%s] %s: %s (%s)\n", mark, cr.Name, cr.Message, cr.Duration.Round(time.Millisecond))
		}
		b.WriteString("\n")
	}

	b.WriteString("Summary\n")
	b.WriteString(strings.Repeat("-", 40))
	b.WriteString("\n")
	fmt.Fprintf(&b, "Hosts:  %d total, %d passed, %d failed\n",
		r.Summary.TotalHosts, r.Summary.PassedHosts, r.Summary.FailedHosts)
	fmt.Fprintf(&b, "Checks: %d total, %d passed, %d failed\n",
		r.Summary.TotalChecks, r.Summary.PassedChecks, r.Summary.FailedChecks)

	return b.String()
}

// RenderMarkdown returns a Markdown-formatted deployment report with tables.
func (r *DeployReport) RenderMarkdown() string {
	var b strings.Builder

	b.WriteString("# Deployment Verification Report\n\n")

	for _, res := range r.Results {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "## %s [%s]\n\n", res.Host, status)
		b.WriteString("| Check | Status | Message | Duration |\n")
		b.WriteString("|-------|--------|---------|----------|\n")
		for _, cr := range res.Checks {
			mark := "PASS"
			if !cr.Passed {
				mark = "FAIL"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				cr.Name, mark, cr.Message, cr.Duration.Round(time.Millisecond))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Total | Passed | Failed |\n")
	b.WriteString("|--------|-------|--------|--------|\n")
	fmt.Fprintf(&b, "| Hosts | %d | %d | %d |\n",
		r.Summary.TotalHosts, r.Summary.PassedHosts, r.Summary.FailedHosts)
	fmt.Fprintf(&b, "| Checks | %d | %d | %d |\n",
		r.Summary.TotalChecks, r.Summary.PassedChecks, r.Summary.FailedChecks)

	return b.String()
}

// jsonReport is the JSON serialization structure for a deployment report.
type jsonReport struct {
	Results     []jsonHostResult `json:"results"`
	Summary     ReportSummary    `json:"summary"`
	GeneratedAt string           `json:"generated_at"`
}

type jsonHostResult struct {
	Host      string            `json:"host"`
	Passed    bool              `json:"passed"`
	Checks    []jsonCheckResult `json:"checks"`
	Timestamp string            `json:"timestamp"`
}

type jsonCheckResult struct {
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
}

// RenderJSON returns a JSON-formatted deployment report suitable for
// CI pipeline consumption.
func (r *DeployReport) RenderJSON() (string, error) {
	jr := jsonReport{
		Summary:     r.Summary,
		GeneratedAt: r.GeneratedAt.UTC().Format(time.RFC3339),
	}

	for _, res := range r.Results {
		hr := jsonHostResult{
			Host:      res.Host,
			Passed:    res.Passed,
			Timestamp: res.Timestamp.UTC().Format(time.RFC3339),
		}
		for _, cr := range res.Checks {
			hr.Checks = append(hr.Checks, jsonCheckResult{
				Name:       cr.Name,
				Passed:     cr.Passed,
				Message:    cr.Message,
				DurationMs: cr.Duration.Milliseconds(),
			})
		}
		jr.Results = append(jr.Results, hr)
	}

	data, err := json.MarshalIndent(jr, "", "  ")
	if err != nil {
		return "", fmt.Errorf("deploy: marshal report: %w", err)
	}
	return string(data), nil
}

// dpComputeSummary aggregates check statistics across all verification results.
func dpComputeSummary(results []VerifyResult) ReportSummary {
	s := ReportSummary{
		TotalHosts: len(results),
	}
	for _, r := range results {
		if r.Passed {
			s.PassedHosts++
		} else {
			s.FailedHosts++
		}
		for _, c := range r.Checks {
			s.TotalChecks++
			if c.Passed {
				s.PassedChecks++
			} else {
				s.FailedChecks++
			}
		}
	}
	return s
}
