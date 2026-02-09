// Package claude provides a collector that queries Anthropic Admin API usage
// data across multiple accounts and calculates costs based on per-model
// pricing tables.
package claude

import "strings"

// ModelPricing holds the per-million-token costs for a given model.
type ModelPricing struct {
	InputPer1M         float64
	OutputPer1M        float64
	CacheCreationPer1M float64
	CacheReadPer1M     float64
}

// pricing maps known model name prefixes to their pricing. When a model is
// not found by exact name, the table is searched by prefix match (longest
// prefix wins). If nothing matches, fallbackPricing is used.
var pricing = map[string]ModelPricing{
	// Opus tier
	"claude-opus-4-6": {
		InputPer1M:         15.0,
		OutputPer1M:        75.0,
		CacheCreationPer1M: 18.75,
		CacheReadPer1M:     1.50,
	},
	// Sonnet tier
	"claude-sonnet-4-5": {
		InputPer1M:         3.0,
		OutputPer1M:        15.0,
		CacheCreationPer1M: 3.75,
		CacheReadPer1M:     0.30,
	},
	"claude-sonnet-4-0": {
		InputPer1M:         3.0,
		OutputPer1M:        15.0,
		CacheCreationPer1M: 3.75,
		CacheReadPer1M:     0.30,
	},
	"claude-3-5-sonnet": {
		InputPer1M:         3.0,
		OutputPer1M:        15.0,
		CacheCreationPer1M: 3.75,
		CacheReadPer1M:     0.30,
	},
	// Haiku tier
	"claude-haiku-4-5": {
		InputPer1M:         0.80,
		OutputPer1M:        4.0,
		CacheCreationPer1M: 1.0,
		CacheReadPer1M:     0.08,
	},
	"claude-3-5-haiku": {
		InputPer1M:         0.80,
		OutputPer1M:        4.0,
		CacheCreationPer1M: 1.0,
		CacheReadPer1M:     0.08,
	},
	"claude-3-haiku": {
		InputPer1M:         0.25,
		OutputPer1M:        1.25,
		CacheCreationPer1M: 0.30,
		CacheReadPer1M:     0.03,
	},
	// Opus 3 (legacy)
	"claude-3-opus": {
		InputPer1M:         15.0,
		OutputPer1M:        75.0,
		CacheCreationPer1M: 18.75,
		CacheReadPer1M:     1.50,
	},
}

// fallbackPricing is used when a model cannot be matched to any known entry.
// It uses Sonnet-tier pricing as a conservative middle estimate.
var fallbackPricing = ModelPricing{
	InputPer1M:         3.0,
	OutputPer1M:        15.0,
	CacheCreationPer1M: 3.75,
	CacheReadPer1M:     0.30,
}

// LookupPricing returns the pricing for a model name. It first tries an
// exact match, then the longest prefix match, and finally returns the
// fallback pricing.
func LookupPricing(model string) ModelPricing {
	if p, ok := pricing[model]; ok {
		return p
	}

	// Longest-prefix match: e.g. "claude-sonnet-4-5-20250929" matches
	// "claude-sonnet-4-5".
	bestLen := 0
	var bestPricing ModelPricing
	found := false
	for prefix, p := range pricing {
		if strings.HasPrefix(model, prefix) && len(prefix) > bestLen {
			bestLen = len(prefix)
			bestPricing = p
			found = true
		}
	}
	if found {
		return bestPricing
	}

	return fallbackPricing
}

// CalculateCost computes the dollar cost for a given model's token usage.
func CalculateCost(model string, inputTokens, outputTokens, cacheCreation, cacheRead int64) float64 {
	p := LookupPricing(model)

	cost := float64(inputTokens) / 1_000_000.0 * p.InputPer1M
	cost += float64(outputTokens) / 1_000_000.0 * p.OutputPer1M
	cost += float64(cacheCreation) / 1_000_000.0 * p.CacheCreationPer1M
	cost += float64(cacheRead) / 1_000_000.0 * p.CacheReadPer1M

	return cost
}
