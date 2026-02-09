// Package data provides a Structure-of-Arrays time-series store inspired by
// bottom (btm). Instead of storing []struct{Time, CPU, RAM, Disk}, separate
// slices share a common time axis. This is more cache-friendly for rendering
// (iterate one metric at a time) and allows different metrics to have different
// retention policies.
package data

import (
	"sort"
	"sync"
	"time"
)

// StoreConfig controls the behavior of a Store instance.
type StoreConfig struct {
	// DefaultRetention is how long data points are kept before pruning.
	// Zero means 10 minutes.
	DefaultRetention time.Duration

	// MaxPoints is the upper bound on points per series. Zero means 600
	// (10 minutes at 1-second intervals).
	MaxPoints int

	// PruneInterval controls how often automatic pruning runs when using
	// a background goroutine. Zero means 30 seconds.
	PruneInterval time.Duration
}

func (c StoreConfig) defaults() StoreConfig {
	if c.DefaultRetention == 0 {
		c.DefaultRetention = 10 * time.Minute
	}
	if c.MaxPoints == 0 {
		c.MaxPoints = 600
	}
	if c.PruneInterval == 0 {
		c.PruneInterval = 30 * time.Second
	}
	return c
}

// Series is a single named time-series with timestamps and corresponding
// values stored in parallel slices.
type Series struct {
	Name      string
	Times     []time.Time
	Values    []float64
	Retention time.Duration       // per-series override; 0 = use store default
	Labels    map[string]string   // metadata labels
}

// SeriesSnapshot is an immutable copy of a series, safe for concurrent reads
// without holding a lock. Slices are always copied from internal storage.
type SeriesSnapshot struct {
	Name   string
	Times  []time.Time
	Values []float64
	Labels map[string]string
}

// Len returns the number of data points.
func (s *SeriesSnapshot) Len() int {
	if s == nil {
		return 0
	}
	return len(s.Values)
}

// Min returns the minimum value. Returns 0 for empty snapshots.
func (s *SeriesSnapshot) Min() float64 {
	if s == nil || len(s.Values) == 0 {
		return 0
	}
	m := s.Values[0]
	for _, v := range s.Values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// Max returns the maximum value. Returns 0 for empty snapshots.
func (s *SeriesSnapshot) Max() float64 {
	if s == nil || len(s.Values) == 0 {
		return 0
	}
	m := s.Values[0]
	for _, v := range s.Values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// Last returns the most recent value. Returns 0 for empty snapshots.
func (s *SeriesSnapshot) Last() float64 {
	if s == nil || len(s.Values) == 0 {
		return 0
	}
	return s.Values[len(s.Values)-1]
}

// Avg returns the arithmetic mean. Returns 0 for empty snapshots.
func (s *SeriesSnapshot) Avg() float64 {
	if s == nil || len(s.Values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range s.Values {
		sum += v
	}
	return sum / float64(len(s.Values))
}

// Store is the main container for all time-series data. It is safe for
// concurrent use by multiple goroutines.
type Store struct {
	mu     sync.RWMutex
	cfg    StoreConfig
	series map[string]*Series

	// Freeze state: tracked per-series with reference counting.
	frozen map[string]*frozenState

	// Prune stats from the last prune cycle.
	lastPruneStats PruneStats
}

// NewStore creates a new time-series store with the given configuration.
func NewStore(cfg StoreConfig) *Store {
	cfg = cfg.defaults()
	return &Store{
		cfg:    cfg,
		series: make(map[string]*Series),
		frozen: make(map[string]*frozenState),
	}
}

// AddPoint appends a single data point to the named series. If the series
// does not exist, it is created automatically.
func (s *Store) AddPoint(name string, t time.Time, v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If the series is frozen, buffer the point.
	if fs, ok := s.frozen[name]; ok && fs.count > 0 {
		fs.pendingTimes = append(fs.pendingTimes, t)
		fs.pendingValues = append(fs.pendingValues, v)
		return
	}

	ser := s.getOrCreate(name)
	ser.Times = append(ser.Times, t)
	ser.Values = append(ser.Values, v)
	s.enforceMaxPoints(ser)
}

// AddPoints appends multiple data points to the named series in bulk.
func (s *Store) AddPoints(name string, times []time.Time, values []float64) {
	if len(times) != len(values) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if fs, ok := s.frozen[name]; ok && fs.count > 0 {
		fs.pendingTimes = append(fs.pendingTimes, times...)
		fs.pendingValues = append(fs.pendingValues, values...)
		return
	}

	ser := s.getOrCreate(name)
	ser.Times = append(ser.Times, times...)
	ser.Values = append(ser.Values, values...)
	s.enforceMaxPoints(ser)
}

// GetSeries returns a read-only snapshot of the named series. If the series
// is frozen, the frozen snapshot is returned instead of live data.
func (s *Store) GetSeries(name string) (*SeriesSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fs, ok := s.frozen[name]; ok && fs.count > 0 {
		snap := copySnapshot(fs.snapshot)
		return snap, true
	}

	ser, ok := s.series[name]
	if !ok {
		return nil, false
	}
	return snapshotFrom(ser), true
}

// GetRange returns a snapshot containing only points within [start, end].
func (s *Store) GetRange(name string, start, end time.Time) (*SeriesSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ser *Series
	if fs, ok := s.frozen[name]; ok && fs.count > 0 {
		// Build a temporary series from the frozen snapshot for range filtering.
		ser = &Series{
			Name:   fs.snapshot.Name,
			Times:  fs.snapshot.Times,
			Values: fs.snapshot.Values,
			Labels: fs.snapshot.Labels,
		}
	} else {
		var exists bool
		ser, exists = s.series[name]
		if !exists {
			return nil, false
		}
	}

	// Binary search for start index.
	lo := sort.Search(len(ser.Times), func(i int) bool {
		return !ser.Times[i].Before(start)
	})
	// Binary search for end index (exclusive).
	hi := sort.Search(len(ser.Times), func(i int) bool {
		return ser.Times[i].After(end)
	})

	if lo >= hi {
		return &SeriesSnapshot{
			Name:   ser.Name,
			Labels: copyLabels(ser.Labels),
		}, true
	}

	return &SeriesSnapshot{
		Name:   ser.Name,
		Times:  copyTimes(ser.Times[lo:hi]),
		Values: copyValues(ser.Values[lo:hi]),
		Labels: copyLabels(ser.Labels),
	}, true
}

// GetLatest returns the most recent time and value for the named series.
func (s *Store) GetLatest(name string) (time.Time, float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fs, ok := s.frozen[name]; ok && fs.count > 0 {
		snap := fs.snapshot
		if len(snap.Values) == 0 {
			return time.Time{}, 0, false
		}
		n := len(snap.Values)
		return snap.Times[n-1], snap.Values[n-1], true
	}

	ser, ok := s.series[name]
	if !ok || len(ser.Values) == 0 {
		return time.Time{}, 0, false
	}
	n := len(ser.Values)
	return ser.Times[n-1], ser.Values[n-1], true
}

// GetLatestN returns a snapshot of the last n points for the named series.
func (s *Store) GetLatestN(name string, n int) (*SeriesSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var times []time.Time
	var values []float64
	var labels map[string]string
	var sname string

	if fs, ok := s.frozen[name]; ok && fs.count > 0 {
		times = fs.snapshot.Times
		values = fs.snapshot.Values
		labels = fs.snapshot.Labels
		sname = fs.snapshot.Name
	} else {
		ser, exists := s.series[name]
		if !exists {
			return nil, false
		}
		times = ser.Times
		values = ser.Values
		labels = ser.Labels
		sname = ser.Name
	}

	if n > len(values) {
		n = len(values)
	}
	start := len(values) - n

	return &SeriesSnapshot{
		Name:   sname,
		Times:  copyTimes(times[start:]),
		Values: copyValues(values[start:]),
		Labels: copyLabels(labels),
	}, true
}

// ListSeries returns the names of all series in sorted order.
func (s *Store) ListSeries() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.series))
	for name := range s.series {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DeleteSeries removes a series entirely, including any frozen state.
func (s *Store) DeleteSeries(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.series, name)
	delete(s.frozen, name)
}

// SetRetention overrides the retention duration for a specific series.
// A zero duration reverts to the store default.
func (s *Store) SetRetention(name string, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ser := s.getOrCreate(name)
	ser.Retention = d
}

// SetLabels sets metadata labels on a series.
func (s *Store) SetLabels(name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ser := s.getOrCreate(name)
	ser.Labels = copyLabels(labels)
}

// retentionFor returns the effective retention for a series.
func (s *Store) retentionFor(ser *Series) time.Duration {
	if ser.Retention > 0 {
		return ser.Retention
	}
	return s.cfg.DefaultRetention
}

// getOrCreate returns the series with the given name, creating it if needed.
// Must be called with the write lock held.
func (s *Store) getOrCreate(name string) *Series {
	ser, ok := s.series[name]
	if !ok {
		ser = &Series{
			Name:   name,
			Labels: make(map[string]string),
		}
		s.series[name] = ser
	}
	return ser
}

// enforceMaxPoints trims the oldest points if the series exceeds MaxPoints.
func (s *Store) enforceMaxPoints(ser *Series) {
	if len(ser.Values) > s.cfg.MaxPoints {
		excess := len(ser.Values) - s.cfg.MaxPoints
		ser.Times = ser.Times[excess:]
		ser.Values = ser.Values[excess:]
	}
}

// snapshotFrom creates an immutable copy from a live series.
func snapshotFrom(ser *Series) *SeriesSnapshot {
	return &SeriesSnapshot{
		Name:   ser.Name,
		Times:  copyTimes(ser.Times),
		Values: copyValues(ser.Values),
		Labels: copyLabels(ser.Labels),
	}
}

// copySnapshot duplicates a snapshot.
func copySnapshot(snap *SeriesSnapshot) *SeriesSnapshot {
	return &SeriesSnapshot{
		Name:   snap.Name,
		Times:  copyTimes(snap.Times),
		Values: copyValues(snap.Values),
		Labels: copyLabels(snap.Labels),
	}
}

func copyTimes(src []time.Time) []time.Time {
	if len(src) == 0 {
		return nil
	}
	dst := make([]time.Time, len(src))
	copy(dst, src)
	return dst
}

func copyValues(src []float64) []float64 {
	if len(src) == 0 {
		return nil
	}
	dst := make([]float64, len(src))
	copy(dst, src)
	return dst
}

func copyLabels(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

