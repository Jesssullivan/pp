package data

import (
	"sort"
	"time"
)

// PruneStats holds metrics from the most recent prune cycle.
type PruneStats struct {
	PointsRemoved int
	SeriesPruned  int
	Duration      time.Duration
}

// Prune removes data points older than each series' effective retention
// period. It uses binary search on the sorted (append-only) time axis for
// O(log n) cutoff discovery and slice reslicing to avoid unnecessary copies.
func (s *Store) Prune() {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	var stats PruneStats

	for _, ser := range s.series {
		// Skip frozen series entirely -- their live data is stale anyway
		// and the snapshot is immutable.
		if fs, ok := s.frozen[ser.Name]; ok && fs.count > 0 {
			continue
		}

		retention := s.retentionFor(ser)
		cutoff := time.Now().Add(-retention)

		// Binary search: find the first index where time is after cutoff.
		idx := sort.Search(len(ser.Times), func(i int) bool {
			return ser.Times[i].After(cutoff)
		})

		if idx > 0 {
			stats.PointsRemoved += idx
			stats.SeriesPruned++

			// Reslice to avoid copying when possible. If we're removing
			// more than half the data, compact to release memory.
			if idx > len(ser.Times)/2 {
				newTimes := make([]time.Time, len(ser.Times)-idx)
				copy(newTimes, ser.Times[idx:])
				ser.Times = newTimes

				newValues := make([]float64, len(ser.Values)-idx)
				copy(newValues, ser.Values[idx:])
				ser.Values = newValues
			} else {
				ser.Times = ser.Times[idx:]
				ser.Values = ser.Values[idx:]
			}
		}
	}

	stats.Duration = time.Since(start)
	s.lastPruneStats = stats
}

// PruneStats returns the metrics from the most recent Prune call.
func (s *Store) PruneStats() PruneStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastPruneStats
}
