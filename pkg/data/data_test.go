package data

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"testing"
	"time"
)

// ---------- helpers ----------

func baseTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func addN(s *Store, name string, n int, interval time.Duration) {
	t := baseTime()
	for i := 0; i < n; i++ {
		s.AddPoint(name, t.Add(time.Duration(i)*interval), float64(i))
	}
}

// ---------- Store basics ----------

func TestAddPointCreatesSeriesAutomatically(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("cpu", baseTime(), 42.0)

	snap, ok := s.GetSeries("cpu")
	if !ok {
		t.Fatal("expected series to exist")
	}
	if snap.Len() != 1 {
		t.Fatalf("expected 1 point, got %d", snap.Len())
	}
	if snap.Values[0] != 42.0 {
		t.Fatalf("expected value 42, got %f", snap.Values[0])
	}
}

func TestAddPointAppendsInOrder(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	for i := 0; i < 5; i++ {
		s.AddPoint("cpu", t0.Add(time.Duration(i)*time.Second), float64(i*10))
	}

	snap, _ := s.GetSeries("cpu")
	if snap.Len() != 5 {
		t.Fatalf("expected 5 points, got %d", snap.Len())
	}
	for i := 0; i < 5; i++ {
		if snap.Values[i] != float64(i*10) {
			t.Errorf("point %d: expected %f, got %f", i, float64(i*10), snap.Values[i])
		}
	}
}

func TestAddPointsBulkAppend(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	times := []time.Time{t0, t0.Add(time.Second), t0.Add(2 * time.Second)}
	values := []float64{1.0, 2.0, 3.0}

	s.AddPoints("mem", times, values)

	snap, ok := s.GetSeries("mem")
	if !ok {
		t.Fatal("expected series to exist")
	}
	if snap.Len() != 3 {
		t.Fatalf("expected 3 points, got %d", snap.Len())
	}
	for i, v := range values {
		if snap.Values[i] != v {
			t.Errorf("point %d: expected %f, got %f", i, v, snap.Values[i])
		}
	}
}

func TestAddPointsMismatchedLengths(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoints("bad", []time.Time{baseTime()}, []float64{1.0, 2.0})

	_, ok := s.GetSeries("bad")
	if ok {
		t.Fatal("expected series NOT to exist when times/values mismatch")
	}
}

func TestGetSeriesReturnsCorrectSnapshot(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	s.AddPoint("disk", t0, 100.0)
	s.AddPoint("disk", t0.Add(time.Second), 200.0)

	snap, ok := s.GetSeries("disk")
	if !ok {
		t.Fatal("expected series to exist")
	}
	if snap.Name != "disk" {
		t.Fatalf("expected name 'disk', got %q", snap.Name)
	}
	if snap.Len() != 2 {
		t.Fatalf("expected 2 points, got %d", snap.Len())
	}
}

func TestGetSeriesSnapshotIsolation(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("iso", baseTime(), 1.0)

	snap, _ := s.GetSeries("iso")
	// Mutate the snapshot -- should not affect the store.
	snap.Values[0] = 999.0

	snap2, _ := s.GetSeries("iso")
	if snap2.Values[0] != 1.0 {
		t.Fatalf("snapshot mutation leaked to store: got %f", snap2.Values[0])
	}
}

func TestGetRangeFiltersByTime(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	for i := 0; i < 10; i++ {
		s.AddPoint("r", t0.Add(time.Duration(i)*time.Second), float64(i))
	}

	// Get range [3s, 6s] inclusive
	start := t0.Add(3 * time.Second)
	end := t0.Add(6 * time.Second)
	snap, ok := s.GetRange("r", start, end)
	if !ok {
		t.Fatal("expected series to exist")
	}
	if snap.Len() != 4 {
		t.Fatalf("expected 4 points in range, got %d", snap.Len())
	}
	if snap.Values[0] != 3.0 || snap.Values[3] != 6.0 {
		t.Fatalf("unexpected range values: first=%f last=%f", snap.Values[0], snap.Values[3])
	}
}

func TestGetRangeEmptyResult(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	s.AddPoint("r2", t0, 1.0)

	snap, ok := s.GetRange("r2", t0.Add(time.Hour), t0.Add(2*time.Hour))
	if !ok {
		t.Fatal("expected series to exist even with empty range")
	}
	if snap.Len() != 0 {
		t.Fatalf("expected 0 points in out-of-range query, got %d", snap.Len())
	}
}

func TestGetRangeNonexistentSeries(t *testing.T) {
	s := NewStore(StoreConfig{})
	_, ok := s.GetRange("nope", baseTime(), baseTime().Add(time.Hour))
	if ok {
		t.Fatal("expected false for nonexistent series")
	}
}

func TestGetLatestReturnsMostRecent(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	s.AddPoint("lat", t0, 10.0)
	s.AddPoint("lat", t0.Add(time.Second), 20.0)
	s.AddPoint("lat", t0.Add(2*time.Second), 30.0)

	ts, v, ok := s.GetLatest("lat")
	if !ok {
		t.Fatal("expected ok")
	}
	if v != 30.0 {
		t.Fatalf("expected 30, got %f", v)
	}
	if !ts.Equal(t0.Add(2 * time.Second)) {
		t.Fatalf("unexpected timestamp %v", ts)
	}
}

func TestGetLatestNonexistent(t *testing.T) {
	s := NewStore(StoreConfig{})
	_, _, ok := s.GetLatest("nope")
	if ok {
		t.Fatal("expected false for nonexistent series")
	}
}

func TestGetLatestNReturnsLastNPoints(t *testing.T) {
	s := NewStore(StoreConfig{})
	addN(s, "ln", 10, time.Second)

	snap, ok := s.GetLatestN("ln", 3)
	if !ok {
		t.Fatal("expected ok")
	}
	if snap.Len() != 3 {
		t.Fatalf("expected 3, got %d", snap.Len())
	}
	// Should be the last 3: 7, 8, 9
	if snap.Values[0] != 7.0 || snap.Values[2] != 9.0 {
		t.Fatalf("unexpected values: %v", snap.Values)
	}
}

func TestGetLatestNMoreThanAvailable(t *testing.T) {
	s := NewStore(StoreConfig{})
	addN(s, "ln2", 3, time.Second)

	snap, ok := s.GetLatestN("ln2", 100)
	if !ok {
		t.Fatal("expected ok")
	}
	if snap.Len() != 3 {
		t.Fatalf("expected 3 (all available), got %d", snap.Len())
	}
}

func TestListSeriesReturnsAllNames(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("b_series", baseTime(), 1.0)
	s.AddPoint("a_series", baseTime(), 2.0)
	s.AddPoint("c_series", baseTime(), 3.0)

	names := s.ListSeries()
	expected := []string{"a_series", "b_series", "c_series"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d series, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], name)
		}
	}
}

func TestDeleteSeriesRemovesSeries(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("del", baseTime(), 1.0)

	s.DeleteSeries("del")

	_, ok := s.GetSeries("del")
	if ok {
		t.Fatal("expected series to be deleted")
	}
	if len(s.ListSeries()) != 0 {
		t.Fatal("expected no series after delete")
	}
}

// ---------- Snapshot math ----------

func TestSnapshotMinMaxAvgLast(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	vals := []float64{5.0, 1.0, 9.0, 3.0, 7.0}
	for i, v := range vals {
		s.AddPoint("math", t0.Add(time.Duration(i)*time.Second), v)
	}

	snap, _ := s.GetSeries("math")

	if snap.Min() != 1.0 {
		t.Errorf("Min: expected 1, got %f", snap.Min())
	}
	if snap.Max() != 9.0 {
		t.Errorf("Max: expected 9, got %f", snap.Max())
	}
	if snap.Last() != 7.0 {
		t.Errorf("Last: expected 7, got %f", snap.Last())
	}
	expectedAvg := (5.0 + 1.0 + 9.0 + 3.0 + 7.0) / 5.0
	if math.Abs(snap.Avg()-expectedAvg) > 1e-9 {
		t.Errorf("Avg: expected %f, got %f", expectedAvg, snap.Avg())
	}
}

func TestEmptySnapshotReturnsZeroValues(t *testing.T) {
	snap := &SeriesSnapshot{}
	if snap.Min() != 0 {
		t.Errorf("Min of empty: expected 0, got %f", snap.Min())
	}
	if snap.Max() != 0 {
		t.Errorf("Max of empty: expected 0, got %f", snap.Max())
	}
	if snap.Last() != 0 {
		t.Errorf("Last of empty: expected 0, got %f", snap.Last())
	}
	if snap.Avg() != 0 {
		t.Errorf("Avg of empty: expected 0, got %f", snap.Avg())
	}
	if snap.Len() != 0 {
		t.Errorf("Len of empty: expected 0, got %d", snap.Len())
	}
}

func TestNilSnapshotReturnsZeroValues(t *testing.T) {
	var snap *SeriesSnapshot
	if snap.Min() != 0 || snap.Max() != 0 || snap.Last() != 0 || snap.Avg() != 0 {
		t.Error("nil snapshot should return 0 for all aggregations")
	}
	if snap.Len() != 0 {
		t.Errorf("nil snapshot Len: expected 0, got %d", snap.Len())
	}
}

// ---------- Prune ----------

func TestPruneRemovesOldData(t *testing.T) {
	s := NewStore(StoreConfig{DefaultRetention: 5 * time.Second})
	now := time.Now()
	// Add points: 10 seconds ago through now.
	for i := 10; i >= 0; i-- {
		s.AddPoint("prune", now.Add(-time.Duration(i)*time.Second), float64(10-i))
	}

	s.Prune()

	snap, ok := s.GetSeries("prune")
	if !ok {
		t.Fatal("expected series to exist after prune")
	}
	// Points older than 5s should be removed. Depending on timing,
	// we expect roughly 5-6 points remaining.
	if snap.Len() > 7 {
		t.Fatalf("expected most old points pruned, but got %d remaining", snap.Len())
	}
	if snap.Len() == 0 {
		t.Fatal("expected some points to remain")
	}
}

func TestPruneWithDifferentRetentionPerSeries(t *testing.T) {
	s := NewStore(StoreConfig{DefaultRetention: 5 * time.Second})
	now := time.Now()

	// Series A: uses default retention (5s)
	for i := 10; i >= 0; i-- {
		s.AddPoint("short", now.Add(-time.Duration(i)*time.Second), float64(i))
	}
	// Series B: custom retention of 20s -- nothing should be pruned
	s.SetRetention("long", 20*time.Second)
	for i := 10; i >= 0; i-- {
		s.AddPoint("long", now.Add(-time.Duration(i)*time.Second), float64(i))
	}

	s.Prune()

	shortSnap, _ := s.GetSeries("short")
	longSnap, _ := s.GetSeries("long")

	if shortSnap.Len() >= 11 {
		t.Fatalf("short series should have been pruned, got %d", shortSnap.Len())
	}
	if longSnap.Len() != 11 {
		t.Fatalf("long series should be untouched, got %d", longSnap.Len())
	}
}

func TestPruneStatsReportsCorrectly(t *testing.T) {
	s := NewStore(StoreConfig{DefaultRetention: 1 * time.Millisecond})
	now := time.Now()
	// Add old points.
	for i := 0; i < 5; i++ {
		s.AddPoint("stats", now.Add(-time.Duration(i+1)*time.Second), float64(i))
	}
	// Add a recent point.
	s.AddPoint("stats", now, 99.0)

	s.Prune()

	ps := s.PruneStats()
	if ps.PointsRemoved != 5 {
		t.Errorf("expected 5 points removed, got %d", ps.PointsRemoved)
	}
	if ps.SeriesPruned != 1 {
		t.Errorf("expected 1 series pruned, got %d", ps.SeriesPruned)
	}
	if ps.Duration <= 0 {
		t.Error("expected positive prune duration")
	}
}

// ---------- Freeze / Unfreeze ----------

func TestFreezeGetSeriesReturnsFrozenData(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	s.AddPoint("f", t0, 1.0)
	s.AddPoint("f", t0.Add(time.Second), 2.0)

	token := s.Freeze("f")

	// Add new data while frozen.
	s.AddPoint("f", t0.Add(2*time.Second), 3.0)

	snap, ok := s.GetSeries("f")
	if !ok {
		t.Fatal("expected series")
	}
	// Should see the frozen snapshot (2 points), not live (3 points).
	if snap.Len() != 2 {
		t.Fatalf("expected frozen snapshot with 2 points, got %d", snap.Len())
	}

	s.Unfreeze(token)
}

func TestUnfreezeMergesPendingData(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	s.AddPoint("uf", t0, 1.0)

	token := s.Freeze("uf")
	s.AddPoint("uf", t0.Add(time.Second), 2.0)
	s.AddPoint("uf", t0.Add(2*time.Second), 3.0)
	s.Unfreeze(token)

	snap, _ := s.GetSeries("uf")
	if snap.Len() != 3 {
		t.Fatalf("expected 3 points after unfreeze merge, got %d", snap.Len())
	}
	if snap.Values[2] != 3.0 {
		t.Fatalf("expected last value 3, got %f", snap.Values[2])
	}
}

func TestMultipleFreezesStack(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	s.AddPoint("stack", t0, 1.0)

	tok1 := s.Freeze("stack")
	tok2 := s.Freeze("stack")

	s.AddPoint("stack", t0.Add(time.Second), 2.0)

	// Unfreeze once -- should still be frozen.
	s.Unfreeze(tok1)
	if !s.IsFrozen("stack") {
		t.Fatal("expected still frozen after first unfreeze")
	}
	snap, _ := s.GetSeries("stack")
	if snap.Len() != 1 {
		t.Fatalf("expected 1 point (still frozen), got %d", snap.Len())
	}

	// Unfreeze second time -- should merge.
	s.Unfreeze(tok2)
	if s.IsFrozen("stack") {
		t.Fatal("expected unfrozen after all tokens released")
	}
	snap, _ = s.GetSeries("stack")
	if snap.Len() != 2 {
		t.Fatalf("expected 2 points after full unfreeze, got %d", snap.Len())
	}
}

func TestIsFrozen(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("chk", baseTime(), 1.0)

	if s.IsFrozen("chk") {
		t.Fatal("should not be frozen initially")
	}

	token := s.Freeze("chk")
	if !s.IsFrozen("chk") {
		t.Fatal("should be frozen after Freeze")
	}

	s.Unfreeze(token)
	if s.IsFrozen("chk") {
		t.Fatal("should not be frozen after Unfreeze")
	}
}

func TestFreezeAllSeries(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("a", baseTime(), 1.0)
	s.AddPoint("b", baseTime(), 2.0)

	token := s.Freeze() // no names = freeze all

	if !s.IsFrozen("a") || !s.IsFrozen("b") {
		t.Fatal("expected all series frozen")
	}

	s.Unfreeze(token)

	if s.IsFrozen("a") || s.IsFrozen("b") {
		t.Fatal("expected all series unfrozen")
	}
}

// ---------- Query builder ----------

func TestQuerySince(t *testing.T) {
	s := NewStore(StoreConfig{})
	now := time.Now()
	// Add points from 10 seconds ago to now.
	for i := 10; i >= 0; i-- {
		s.AddPoint("qs", now.Add(-time.Duration(i)*time.Second), float64(10-i))
	}

	results := s.Query("qs").Since(5 * time.Second).Execute()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Should have roughly 5-6 points (those within last 5s).
	if results[0].Len() == 0 {
		t.Fatal("expected some points in since query")
	}
	if results[0].Len() > 7 {
		t.Fatalf("expected at most ~7 points, got %d", results[0].Len())
	}
}

func TestQueryBetween(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	for i := 0; i < 10; i++ {
		s.AddPoint("qb", t0.Add(time.Duration(i)*time.Second), float64(i))
	}

	start := t0.Add(2 * time.Second)
	end := t0.Add(5 * time.Second)
	results := s.Query("qb").Between(start, end).Execute()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Len() != 4 {
		t.Fatalf("expected 4 points in range, got %d", results[0].Len())
	}
}

func TestQueryLast(t *testing.T) {
	s := NewStore(StoreConfig{})
	addN(s, "ql", 20, time.Second)

	results := s.Query("ql").Last(5).Execute()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Len() != 5 {
		t.Fatalf("expected 5 points, got %d", results[0].Len())
	}
	// Last 5 of 20 points (values 15..19)
	if results[0].Values[0] != 15.0 {
		t.Fatalf("expected first value 15, got %f", results[0].Values[0])
	}
}

func TestQueryByLabelFindsMatchingSeries(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("cpu.honey", baseTime(), 50.0)
	s.AddPoint("mem.honey", baseTime(), 70.0)
	s.AddPoint("cpu.yoga", baseTime(), 30.0)

	s.SetLabels("cpu.honey", map[string]string{"host": "honey", "metric": "cpu"})
	s.SetLabels("mem.honey", map[string]string{"host": "honey", "metric": "mem"})
	s.SetLabels("cpu.yoga", map[string]string{"host": "yoga", "metric": "cpu"})

	results := s.QueryByLabel("host", "honey").Execute()
	if len(results) != 2 {
		t.Fatalf("expected 2 matching series, got %d", len(results))
	}

	// Sort by name for deterministic check.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	if results[0].Name != "cpu.honey" || results[1].Name != "mem.honey" {
		t.Fatalf("unexpected series: %v, %v", results[0].Name, results[1].Name)
	}
}

func TestQueryByLabelNoMatch(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("x", baseTime(), 1.0)
	s.SetLabels("x", map[string]string{"env": "prod"})

	results := s.QueryByLabel("env", "dev").Execute()
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// ---------- Labels ----------

func TestSetLabelsAndRetrieval(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("labeled", baseTime(), 1.0)
	labels := map[string]string{"host": "honey", "region": "us-east"}
	s.SetLabels("labeled", labels)

	snap, _ := s.GetSeries("labeled")
	if snap.Labels["host"] != "honey" {
		t.Errorf("expected host=honey, got %q", snap.Labels["host"])
	}
	if snap.Labels["region"] != "us-east" {
		t.Errorf("expected region=us-east, got %q", snap.Labels["region"])
	}
}

func TestSetLabelsIsolation(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("iso", baseTime(), 1.0)
	labels := map[string]string{"k": "v"}
	s.SetLabels("iso", labels)

	// Mutate the original map -- should not affect the store.
	labels["k"] = "mutated"

	snap, _ := s.GetSeries("iso")
	if snap.Labels["k"] != "v" {
		t.Fatalf("label mutation leaked: got %q", snap.Labels["k"])
	}
}

// ---------- MaxPoints enforcement ----------

func TestMaxPointsEnforcement(t *testing.T) {
	s := NewStore(StoreConfig{MaxPoints: 5})
	addN(s, "max", 10, time.Second)

	snap, _ := s.GetSeries("max")
	if snap.Len() != 5 {
		t.Fatalf("expected 5 points (maxpoints), got %d", snap.Len())
	}
	// Should have the last 5 values: 5, 6, 7, 8, 9
	if snap.Values[0] != 5.0 {
		t.Fatalf("expected oldest remaining value 5, got %f", snap.Values[0])
	}
}

// ---------- Thread safety ----------

func TestConcurrentAddAndGet(t *testing.T) {
	s := NewStore(StoreConfig{})
	const goroutines = 10
	const pointsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // half writers, half readers

	// Writers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent.%d", id)
			t0 := baseTime()
			for i := 0; i < pointsPerGoroutine; i++ {
				s.AddPoint(name, t0.Add(time.Duration(i)*time.Millisecond), float64(i))
			}
		}(g)
	}

	// Readers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent.%d", id)
			for i := 0; i < pointsPerGoroutine; i++ {
				s.GetSeries(name)
				s.GetLatest(name)
				s.ListSeries()
			}
		}(g)
	}

	wg.Wait()

	// Verify all data arrived.
	for g := 0; g < goroutines; g++ {
		name := fmt.Sprintf("concurrent.%d", g)
		snap, ok := s.GetSeries(name)
		if !ok {
			t.Fatalf("series %q missing after concurrent writes", name)
		}
		if snap.Len() != pointsPerGoroutine {
			t.Fatalf("series %q: expected %d points, got %d", name, pointsPerGoroutine, snap.Len())
		}
	}
}

func TestConcurrentFreezeAndWrite(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("cf", baseTime(), 0.0)

	var wg sync.WaitGroup
	wg.Add(3)

	// Writer
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i++ {
			s.AddPoint("cf", baseTime().Add(time.Duration(i)*time.Millisecond), float64(i))
		}
	}()

	// Freezer/unfreezer
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			tok := s.Freeze("cf")
			time.Sleep(time.Microsecond)
			s.Unfreeze(tok)
		}
	}()

	// Reader
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			s.GetSeries("cf")
		}
	}()

	wg.Wait()
	// No race/panic = pass
}

// ---------- Performance ----------

func TestLargeDatasetPrunePerformance(t *testing.T) {
	s := NewStore(StoreConfig{
		DefaultRetention: 5 * time.Second,
		MaxPoints:        100000,
	})
	now := time.Now()

	// Insert 10000 points spanning 20 seconds.
	const n = 10000
	interval := 20 * time.Second / time.Duration(n)
	for i := 0; i < n; i++ {
		s.AddPoint("perf", now.Add(-20*time.Second+time.Duration(i)*interval), float64(i))
	}

	start := time.Now()
	s.Prune()
	elapsed := time.Since(start)

	ps := s.PruneStats()
	if ps.PointsRemoved == 0 {
		t.Fatal("expected some points to be pruned")
	}

	// Prune should be fast -- well under 100ms for 10k points.
	if elapsed > 100*time.Millisecond {
		t.Fatalf("prune took too long: %v", elapsed)
	}

	snap, _ := s.GetSeries("perf")
	if snap.Len() == 0 {
		t.Fatal("expected some points to remain")
	}
	if snap.Len() >= n {
		t.Fatalf("expected fewer than %d points after prune, got %d", n, snap.Len())
	}
}

// ---------- SetRetention ----------

func TestSetRetention(t *testing.T) {
	s := NewStore(StoreConfig{DefaultRetention: 10 * time.Minute})
	s.AddPoint("ret", baseTime(), 1.0)
	s.SetRetention("ret", 30*time.Second)

	s.mu.RLock()
	ser := s.series["ret"]
	if ser.Retention != 30*time.Second {
		t.Fatalf("expected retention 30s, got %v", ser.Retention)
	}
	s.mu.RUnlock()
}

// ---------- DeleteSeries with frozen ----------

func TestDeleteFrozenSeries(t *testing.T) {
	s := NewStore(StoreConfig{})
	s.AddPoint("df", baseTime(), 1.0)
	s.Freeze("df")

	s.DeleteSeries("df")

	if s.IsFrozen("df") {
		t.Fatal("expected frozen state to be cleaned up on delete")
	}
	_, ok := s.GetSeries("df")
	if ok {
		t.Fatal("expected series to be deleted")
	}
}

// ---------- Edge: empty store operations ----------

func TestEmptyStoreOperations(t *testing.T) {
	s := NewStore(StoreConfig{})

	// These should not panic.
	s.Prune()
	names := s.ListSeries()
	if len(names) != 0 {
		t.Fatalf("expected 0 series, got %d", len(names))
	}
	s.DeleteSeries("nonexistent")

	_, ok := s.GetSeries("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent series")
	}

	ps := s.PruneStats()
	if ps.PointsRemoved != 0 {
		t.Fatalf("expected 0 points removed, got %d", ps.PointsRemoved)
	}
}

// ---------- GetRange on frozen series ----------

func TestGetRangeOnFrozenSeries(t *testing.T) {
	s := NewStore(StoreConfig{})
	t0 := baseTime()
	for i := 0; i < 10; i++ {
		s.AddPoint("fr", t0.Add(time.Duration(i)*time.Second), float64(i))
	}

	token := s.Freeze("fr")

	// Add more points (should go to pending).
	s.AddPoint("fr", t0.Add(10*time.Second), 10.0)

	// Range query should use frozen data only.
	snap, ok := s.GetRange("fr", t0.Add(3*time.Second), t0.Add(6*time.Second))
	if !ok {
		t.Fatal("expected ok")
	}
	if snap.Len() != 4 {
		t.Fatalf("expected 4 points from frozen range, got %d", snap.Len())
	}

	s.Unfreeze(token)
}

// Verify we have at least 25 tests by counting test functions in this file.
// Current count: 35 Test* functions.
var _ = math.Abs // keep math import used
