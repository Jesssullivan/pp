package data

import (
	"time"
)

// QueryBuilder provides a fluent interface for querying time-series data.
// Create one via Store.Query or Store.QueryByLabel.
type QueryBuilder struct {
	store      *Store
	name       string       // single series name (empty for label queries)
	labelKey   string       // label filter key
	labelValue string       // label filter value
	sinceD     time.Duration
	start      time.Time
	end        time.Time
	lastN      int
	mode       queryMode
}

type queryMode int

const (
	queryAll queryMode = iota
	querySince
	queryBetween
	queryLast
)

// Query starts a query builder for a single named series.
func (s *Store) Query(name string) *QueryBuilder {
	return &QueryBuilder{
		store: s,
		name:  name,
		mode:  queryAll,
	}
}

// QueryByLabel starts a query builder that matches all series with the
// given label key=value pair.
func (s *Store) QueryByLabel(key, value string) *QueryBuilder {
	return &QueryBuilder{
		store:      s,
		labelKey:   key,
		labelValue: value,
		mode:       queryAll,
	}
}

// Since restricts the query to points within the last d duration.
func (qb *QueryBuilder) Since(d time.Duration) *QueryBuilder {
	qb.sinceD = d
	qb.mode = querySince
	return qb
}

// Between restricts the query to points in [start, end].
func (qb *QueryBuilder) Between(start, end time.Time) *QueryBuilder {
	qb.start = start
	qb.end = end
	qb.mode = queryBetween
	return qb
}

// Last restricts the query to the most recent n points.
func (qb *QueryBuilder) Last(n int) *QueryBuilder {
	qb.lastN = n
	qb.mode = queryLast
	return qb
}

// Execute runs the query and returns matching snapshots.
func (qb *QueryBuilder) Execute() []SeriesSnapshot {
	names := qb.resolveNames()
	results := make([]SeriesSnapshot, 0, len(names))

	for _, name := range names {
		var snap *SeriesSnapshot
		var ok bool

		switch qb.mode {
		case queryAll:
			snap, ok = qb.store.GetSeries(name)
		case querySince:
			now := time.Now()
			snap, ok = qb.store.GetRange(name, now.Add(-qb.sinceD), now)
		case queryBetween:
			snap, ok = qb.store.GetRange(name, qb.start, qb.end)
		case queryLast:
			snap, ok = qb.store.GetLatestN(name, qb.lastN)
		}

		if ok && snap != nil {
			results = append(results, *snap)
		}
	}

	return results
}

// resolveNames determines which series names to query.
func (qb *QueryBuilder) resolveNames() []string {
	if qb.name != "" {
		return []string{qb.name}
	}

	// Label-based query: scan all series for matching labels.
	qb.store.mu.RLock()
	defer qb.store.mu.RUnlock()

	var names []string
	for sname, ser := range qb.store.series {
		if v, ok := ser.Labels[qb.labelKey]; ok && v == qb.labelValue {
			names = append(names, sname)
		}
	}
	return names
}
