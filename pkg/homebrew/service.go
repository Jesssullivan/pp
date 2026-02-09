package homebrew

import (
	"bytes"
	"text/template"
)

// ServiceConfig holds configuration for macOS launchd service integration.
type ServiceConfig struct {
	// BinaryPath is the absolute path to the daemon binary.
	BinaryPath string

	// LogDir is the directory for stdout/stderr log files.
	LogDir string

	// RunDir is the working directory for the daemon process.
	RunDir string

	// KeepAlive controls whether launchd restarts the process on exit.
	KeepAlive bool

	// Interval is the periodic execution interval in seconds (0 = continuous).
	Interval int
}

// GenerateLaunchdPlist renders a macOS launchd plist XML document.
func GenerateLaunchdPlist(config *ServiceConfig) (string, error) {
	tmpl, err := template.New("plist").Parse(hbPlistTemplate())
	if err != nil {
		return "", err
	}

	data := struct {
		*ServiceConfig
		Label string
	}{
		ServiceConfig: config,
		Label:         "com.tinyland.prompt-pulse",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GenerateBrewService renders a Homebrew service do...end Ruby block.
func GenerateBrewService(config *ServiceConfig) (string, error) {
	tmpl, err := template.New("service").Parse(hbBrewServiceTemplate())
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// hbPlistTemplate returns the launchd plist XML template.
func hbPlistTemplate() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{{ .Label }}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{ .BinaryPath }}</string>
    <string>daemon</string>
  </array>
  <key>KeepAlive</key>
  <{{ if .KeepAlive }}true{{ else }}false{{ end }}/>
  <key>StandardOutPath</key>
  <string>{{ .LogDir }}/prompt-pulse.log</string>
  <key>StandardErrorPath</key>
  <string>{{ .LogDir }}/prompt-pulse-error.log</string>
  <key>WorkingDirectory</key>
  <string>{{ .RunDir }}</string>
{{ if gt .Interval 0 }}  <key>StartInterval</key>
  <integer>{{ .Interval }}</integer>
{{ end }}</dict>
</plist>
`
}

// hbBrewServiceTemplate returns the Ruby template for a Homebrew service block.
func hbBrewServiceTemplate() string {
	return `  service do
    run [opt_bin/"prompt-pulse", "daemon"]
    keep_alive {{ if .KeepAlive }}true{{ else }}false{{ end }}
    log_path var/"log/prompt-pulse.log"
    error_log_path var/"log/prompt-pulse-error.log"
{{ if ne .RunDir "" }}    working_dir "{{ .RunDir }}"
{{ end }}  end
`
}
