package deploy

import (
	"fmt"
	"sort"
)

// RolloutPlan describes an ordered multi-host deployment plan.
type RolloutPlan struct {
	// Hosts lists the individual host rollouts in order.
	Hosts []HostRollout

	// Strategy is "serial" (one at a time) or "parallel" (all at once).
	Strategy string

	// DryRun when true means no changes are applied.
	DryRun bool
}

// HostRollout pairs a host profile with its rollout ordering and checks.
type HostRollout struct {
	// Profile is the host to deploy to.
	Profile *HostProfile

	// Order controls the deployment sequence (lower = earlier).
	Order int

	// PreChecks run before deployment to this host.
	PreChecks []Check

	// PostChecks run after deployment to this host.
	PostChecks []Check
}

// NewRolloutPlan creates a RolloutPlan with the given strategy.
// Valid strategies are "serial" and "parallel".
func NewRolloutPlan(strategy string) *RolloutPlan {
	return &RolloutPlan{
		Strategy: strategy,
	}
}

// AddHost appends a host to the rollout plan at the specified order.
func (r *RolloutPlan) AddHost(profile *HostProfile, order int) {
	hr := HostRollout{
		Profile:    profile,
		Order:      order,
		PreChecks:  dpBuildChecks(profile),
		PostChecks: dpBuildChecks(profile),
	}
	r.Hosts = append(r.Hosts, hr)
	sort.Slice(r.Hosts, func(i, j int) bool {
		return r.Hosts[i].Order < r.Hosts[j].Order
	})
}

// Validate checks the rollout plan for errors and returns a list of
// human-readable problems. An empty slice means the plan is valid.
func (r *RolloutPlan) Validate() []string {
	var problems []string

	if r.Strategy != "serial" && r.Strategy != "parallel" {
		problems = append(problems, fmt.Sprintf("invalid strategy %q (must be serial or parallel)", r.Strategy))
	}

	if len(r.Hosts) == 0 {
		problems = append(problems, "rollout plan has no hosts")
	}

	seen := map[int]string{}
	for _, h := range r.Hosts {
		if h.Profile == nil {
			problems = append(problems, "host rollout has nil profile")
			continue
		}
		if prev, ok := seen[h.Order]; ok {
			problems = append(problems, fmt.Sprintf("duplicate order %d: %s and %s", h.Order, prev, h.Profile.Name))
		}
		seen[h.Order] = h.Profile.Name
	}

	return problems
}

// DefaultRolloutPlan returns the standard 3-host rollout in order of
// increasing risk: xoxd-bates (dev), petting-zoo-mini (streaming),
// honey (production server).
func DefaultRolloutPlan() *RolloutPlan {
	plan := NewRolloutPlan("serial")
	plan.AddHost(XoxdBates(), 1)
	plan.AddHost(PettingZooMini(), 2)
	plan.AddHost(Honey(), 3)
	return plan
}
