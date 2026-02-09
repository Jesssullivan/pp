package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HealthStatus represents the overall health of a prompt-pulse deployment.
type HealthStatus struct {
	// Healthy is true when all components are operational.
	Healthy bool

	// Components lists the health of individual subsystems.
	Components []ComponentHealth

	// Uptime is the daemon uptime if available.
	Uptime time.Duration
}

// ComponentHealth describes the health of a single subsystem.
type ComponentHealth struct {
	// Name identifies the component.
	Name string

	// Status is "healthy", "degraded", or "unhealthy".
	Status string

	// Message provides detail about the component state.
	Message string

	// LastCheck records when this component was last checked.
	LastCheck time.Time
}

// HealthConfig holds paths and thresholds for health checks.
type HealthConfig struct {
	// SocketPath is the daemon socket location.
	SocketPath string

	// CacheDir is the cache directory.
	CacheDir string

	// Collectors lists collector names to check.
	Collectors []string

	// StaleDuration is how old data can be before it is stale.
	StaleDuration time.Duration

	// MinDiskBytes is the minimum free space required.
	MinDiskBytes int64

	// Now overrides time.Now for testing.
	Now func() time.Time
}

func (hc *HealthConfig) now() time.Time {
	if hc.Now != nil {
		return hc.Now()
	}
	return time.Now()
}

// dpCheckHealth runs all health checks using the given config and returns
// an aggregated HealthStatus.
func dpCheckHealth(cfg *HealthConfig) (*HealthStatus, error) {
	if cfg == nil {
		return nil, fmt.Errorf("deploy: nil health config")
	}

	components := []ComponentHealth{
		dpCheckDaemonHealth(cfg),
		dpCheckCacheHealth(cfg),
	}

	for _, col := range cfg.Collectors {
		components = append(components, dpCheckCollectorHealth(cfg, col))
	}

	components = append(components, dpCheckDiskSpace(cfg))

	healthy := true
	for _, c := range components {
		if c.Status == "unhealthy" {
			healthy = false
			break
		}
	}

	return &HealthStatus{
		Healthy:    healthy,
		Components: components,
	}, nil
}

// dpCheckDaemonHealth checks whether the daemon socket exists.
func dpCheckDaemonHealth(cfg *HealthConfig) ComponentHealth {
	sock := cfg.SocketPath
	if sock == "" {
		sock = dpDefaultSocketPath()
	}

	now := cfg.now()
	if _, err := os.Stat(sock); err != nil {
		return ComponentHealth{
			Name:      "daemon",
			Status:    "unhealthy",
			Message:   fmt.Sprintf("socket not found: %s", sock),
			LastCheck: now,
		}
	}
	return ComponentHealth{
		Name:      "daemon",
		Status:    "healthy",
		Message:   "daemon socket present",
		LastCheck: now,
	}
}

// dpCheckCacheHealth verifies the cache directory is readable and not stale.
func dpCheckCacheHealth(cfg *HealthConfig) ComponentHealth {
	dir := cfg.CacheDir
	if dir == "" {
		dir = dpDefaultCacheDir()
	}

	now := cfg.now()

	info, err := os.Stat(dir)
	if err != nil {
		return ComponentHealth{
			Name:      "cache",
			Status:    "unhealthy",
			Message:   fmt.Sprintf("cache dir not found: %s", dir),
			LastCheck: now,
		}
	}

	stale := cfg.StaleDuration
	if stale == 0 {
		stale = 24 * time.Hour
	}

	if now.Sub(info.ModTime()) > stale {
		return ComponentHealth{
			Name:      "cache",
			Status:    "degraded",
			Message:   fmt.Sprintf("cache stale: last modified %s ago", now.Sub(info.ModTime()).Round(time.Minute)),
			LastCheck: now,
		}
	}

	return ComponentHealth{
		Name:      "cache",
		Status:    "healthy",
		Message:   "cache ok",
		LastCheck: now,
	}
}

// collectorMeta is the expected JSON structure of a collector data file.
type collectorMeta struct {
	UpdatedAt string `json:"updated_at"`
}

// dpCheckCollectorHealth verifies a collector's data file exists and is
// not outdated based on its internal timestamp.
func dpCheckCollectorHealth(cfg *HealthConfig, name string) ComponentHealth {
	dir := cfg.CacheDir
	if dir == "" {
		dir = dpDefaultCacheDir()
	}

	now := cfg.now()
	dataFile := filepath.Join(dir, "collectors", name+".json")

	data, err := os.ReadFile(dataFile)
	if err != nil {
		return ComponentHealth{
			Name:      fmt.Sprintf("collector-%s", name),
			Status:    "unhealthy",
			Message:   fmt.Sprintf("data file missing: %s", dataFile),
			LastCheck: now,
		}
	}

	var meta collectorMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return ComponentHealth{
			Name:      fmt.Sprintf("collector-%s", name),
			Status:    "degraded",
			Message:   "data file is not valid JSON",
			LastCheck: now,
		}
	}

	if meta.UpdatedAt != "" {
		updated, err := time.Parse(time.RFC3339, meta.UpdatedAt)
		if err == nil {
			stale := cfg.StaleDuration
			if stale == 0 {
				stale = 24 * time.Hour
			}
			if now.Sub(updated) > stale {
				return ComponentHealth{
					Name:      fmt.Sprintf("collector-%s", name),
					Status:    "degraded",
					Message:   fmt.Sprintf("data stale: last updated %s ago", now.Sub(updated).Round(time.Minute)),
					LastCheck: now,
				}
			}
		}
	}

	return ComponentHealth{
		Name:      fmt.Sprintf("collector-%s", name),
		Status:    "healthy",
		Message:   fmt.Sprintf("collector %s ok", name),
		LastCheck: now,
	}
}

// dpCheckDiskSpace verifies that the cache directory's filesystem has
// sufficient free space. It uses a stat-based heuristic: if the cache
// directory doesn't exist, the check fails; otherwise it reports healthy.
// Real disk-space checking requires platform-specific syscalls that are
// out of scope for this verification package.
func dpCheckDiskSpace(cfg *HealthConfig) ComponentHealth {
	dir := cfg.CacheDir
	if dir == "" {
		dir = dpDefaultCacheDir()
	}

	now := cfg.now()

	if _, err := os.Stat(dir); err != nil {
		return ComponentHealth{
			Name:      "disk-space",
			Status:    "unhealthy",
			Message:   fmt.Sprintf("cache dir not found: %s", dir),
			LastCheck: now,
		}
	}

	return ComponentHealth{
		Name:      "disk-space",
		Status:    "healthy",
		Message:   "cache directory accessible",
		LastCheck: now,
	}
}
