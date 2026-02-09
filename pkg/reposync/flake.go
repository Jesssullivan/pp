package reposync

import (
	"fmt"
	"regexp"
	"strings"
)

// FlakeInput represents a single Nix flake input declaration.
type FlakeInput struct {
	// Name is the input identifier (e.g., "prompt-pulse").
	Name string

	// URL is the flake URL (e.g., "gitlab:tinyland/projects/prompt-pulse").
	URL string

	// Rev is the pinned revision hash, if any.
	Rev string

	// Type is the input type ("gitlab", "github", "path", etc.).
	Type string

	// Flake indicates whether this input is itself a flake.
	Flake bool
}

// rsParseFlakeInputs extracts FlakeInput entries from raw flake.nix content.
// It uses simple regex parsing since we avoid external Nix/TOML libraries.
func rsParseFlakeInputs(flakeContent string) ([]FlakeInput, error) {
	if strings.TrimSpace(flakeContent) == "" {
		return nil, fmt.Errorf("flake content is empty")
	}

	var inputs []FlakeInput

	// Match patterns like:  name.url = "...";
	urlRe := regexp.MustCompile(`(\w[\w-]*)\.url\s*=\s*"([^"]+)"\s*;`)
	// Match patterns like:  name.flake = false;
	flakeRe := regexp.MustCompile(`(\w[\w-]*)\.flake\s*=\s*(true|false)\s*;`)
	// Match patterns like:  name.rev = "abc123";
	// (less common but used for pinning)
	// Also match inputs.name = { url = "..."; flake = false; };
	revRe := regexp.MustCompile(`(\w[\w-]*)\.rev\s*=\s*"([^"]+)"\s*;`)

	// Build a map so we can merge url/flake/rev for the same name.
	type entry struct {
		url   string
		rev   string
		flake bool
		seen  bool
	}
	m := make(map[string]*entry)
	// Track insertion order.
	var order []string

	for _, match := range urlRe.FindAllStringSubmatch(flakeContent, -1) {
		name := match[1]
		if _, ok := m[name]; !ok {
			m[name] = &entry{flake: true} // default flake=true
			order = append(order, name)
		}
		m[name].url = match[2]
		m[name].seen = true
	}

	for _, match := range flakeRe.FindAllStringSubmatch(flakeContent, -1) {
		name := match[1]
		if _, ok := m[name]; !ok {
			m[name] = &entry{flake: true}
			order = append(order, name)
		}
		m[name].flake = match[2] == "true"
	}

	for _, match := range revRe.FindAllStringSubmatch(flakeContent, -1) {
		name := match[1]
		if _, ok := m[name]; !ok {
			m[name] = &entry{flake: true}
			order = append(order, name)
		}
		m[name].rev = match[2]
	}

	for _, name := range order {
		e := m[name]
		if !e.seen {
			continue
		}
		fi := FlakeInput{
			Name:  name,
			URL:   e.url,
			Rev:   e.rev,
			Flake: e.flake,
			Type:  rsInferFlakeType(e.url),
		}
		inputs = append(inputs, fi)
	}

	return inputs, nil
}

// rsUpdateFlakeRev replaces the rev pin for a named input in flake.nix content.
func rsUpdateFlakeRev(flakeContent, inputName, newRev string) (string, error) {
	if strings.TrimSpace(inputName) == "" {
		return "", fmt.Errorf("input name must not be empty")
	}
	if strings.TrimSpace(newRev) == "" {
		return "", fmt.Errorf("new rev must not be empty")
	}

	// Try to replace existing rev declaration.
	revPattern := regexp.MustCompile(
		fmt.Sprintf(`(%s\.rev\s*=\s*")([^"]+)("\s*;)`, regexp.QuoteMeta(inputName)))
	if revPattern.MatchString(flakeContent) {
		result := revPattern.ReplaceAllString(flakeContent, "${1}"+newRev+"${3}")
		return result, nil
	}

	// If no rev line exists, insert one after the url line.
	urlPattern := regexp.MustCompile(
		fmt.Sprintf(`(%s\.url\s*=\s*"[^"]+"\s*;)`, regexp.QuoteMeta(inputName)))
	if urlPattern.MatchString(flakeContent) {
		revLine := fmt.Sprintf("    %s.rev = \"%s\";", inputName, newRev)
		result := urlPattern.ReplaceAllString(flakeContent, "${1}\n"+revLine)
		return result, nil
	}

	return "", fmt.Errorf("input %q not found in flake content", inputName)
}

// rsGenerateFlakeInput creates a FlakeInput for the external sync target.
func rsGenerateFlakeInput(config *SyncConfig) *FlakeInput {
	if config == nil {
		return nil
	}
	// Convert "gitlab.com/group/project" to "gitlab:group/project".
	url := rsRepoToFlakeURL(config.TargetRepo)
	return &FlakeInput{
		Name:  rsRepoShortName(config.TargetRepo),
		URL:   url,
		Flake: true,
		Type:  rsInferFlakeType(url),
	}
}

// rsValidateFlakeInput checks a FlakeInput for common problems.
func rsValidateFlakeInput(input *FlakeInput) []string {
	var errs []string
	if input == nil {
		return []string{"input is nil"}
	}
	if strings.TrimSpace(input.Name) == "" {
		errs = append(errs, "name is required")
	}
	if strings.TrimSpace(input.URL) == "" {
		errs = append(errs, "url is required")
	}
	if strings.TrimSpace(input.Type) == "" {
		errs = append(errs, "type is required")
	}
	// Name must be a valid Nix identifier.
	if input.Name != "" && !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`).MatchString(input.Name) {
		errs = append(errs, fmt.Sprintf("name %q is not a valid Nix identifier", input.Name))
	}
	return errs
}

// rsInferFlakeType guesses the flake type from a URL.
func rsInferFlakeType(url string) string {
	switch {
	case strings.HasPrefix(url, "gitlab:"):
		return "gitlab"
	case strings.HasPrefix(url, "github:"):
		return "github"
	case strings.HasPrefix(url, "path:"):
		return "path"
	case strings.HasPrefix(url, "git+"):
		return "git"
	default:
		return "indirect"
	}
}

// rsRepoToFlakeURL converts a domain/path repo string to flake URL format.
func rsRepoToFlakeURL(repo string) string {
	if strings.HasPrefix(repo, "gitlab.com/") {
		return "gitlab:" + strings.TrimPrefix(repo, "gitlab.com/")
	}
	if strings.HasPrefix(repo, "github.com/") {
		return "github:" + strings.TrimPrefix(repo, "github.com/")
	}
	return "git+https://" + repo
}

// rsRepoShortName extracts the last path segment from a repo URL as an input name.
func rsRepoShortName(repo string) string {
	parts := strings.Split(strings.TrimSuffix(repo, "/"), "/")
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}
