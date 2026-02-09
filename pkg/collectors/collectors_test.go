package collectors

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// --- Registry Tests ---

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()
	c := NewMockCollector("test", time.Second)

	if err := r.Register(c); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, ok := r.Get("test")
	if !ok {
		t.Fatal("Get returned false for registered collector")
	}
	if got.Name() != "test" {
		t.Errorf("Name = %q, want %q", got.Name(), "test")
	}
}

func TestRegistryDuplicateNameError(t *testing.T) {
	r := NewRegistry()
	c1 := NewMockCollector("dup", time.Second)
	c2 := NewMockCollector("dup", time.Second)

	if err := r.Register(c1); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}
	if err := r.Register(c2); err == nil {
		t.Fatal("second Register should have returned an error for duplicate name")
	}
}

func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()
	c := NewMockCollector("gone", time.Second)
	_ = r.Register(c)

	r.Unregister("gone")

	if _, ok := r.Get("gone"); ok {
		t.Fatal("Get returned true after Unregister")
	}
}

func TestRegistryUnregisterNonExistent(t *testing.T) {
	r := NewRegistry()
	// Should not panic.
	r.Unregister("does-not-exist")
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("missing"); ok {
		t.Fatal("Get should return false for unregistered collector")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("charlie", time.Second))
	_ = r.Register(NewMockCollector("alpha", time.Second))
	_ = r.Register(NewMockCollector("bravo", time.Second))

	names := r.List()
	expected := []string{"alpha", "bravo", "charlie"}

	if len(names) != len(expected) {
		t.Fatalf("List returned %d names, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("List[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestRegistryListEmpty(t *testing.T) {
	r := NewRegistry()
	names := r.List()
	if len(names) != 0 {
		t.Fatalf("List returned %d names for empty registry, want 0", len(names))
	}
}

func TestRegistryStatus(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("metrics", time.Second))

	s, ok := r.Status("metrics")
	if !ok {
		t.Fatal("Status returned false for registered collector")
	}
	if s.Name != "metrics" {
		t.Errorf("Status.Name = %q, want %q", s.Name, "metrics")
	}
	if !s.Healthy {
		t.Error("initial status should be healthy")
	}
	if s.RunCount != 0 {
		t.Errorf("initial RunCount = %d, want 0", s.RunCount)
	}
}

func TestRegistryStatusNotFound(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Status("nope"); ok {
		t.Fatal("Status should return false for unregistered collector")
	}
}

func TestRegistryAllStatus(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("b", time.Second))
	_ = r.Register(NewMockCollector("a", time.Second))

	statuses := r.AllStatus()
	if len(statuses) != 2 {
		t.Fatalf("AllStatus returned %d, want 2", len(statuses))
	}
	// Should be sorted.
	if statuses[0].Name != "a" || statuses[1].Name != "b" {
		t.Errorf("AllStatus not sorted: got %q, %q", statuses[0].Name, statuses[1].Name)
	}
}

func TestRegistryAllStatusEmpty(t *testing.T) {
	r := NewRegistry()
	statuses := r.AllStatus()
	if len(statuses) != 0 {
		t.Fatalf("AllStatus returned %d for empty registry, want 0", len(statuses))
	}
}

func TestRegistryStatusAfterUnregister(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("temp", time.Second))
	r.Unregister("temp")

	if _, ok := r.Status("temp"); ok {
		t.Fatal("Status should return false after Unregister")
	}
	statuses := r.AllStatus()
	if len(statuses) != 0 {
		t.Fatalf("AllStatus should be empty after Unregister, got %d", len(statuses))
	}
}

// --- Mock Collector Tests ---

func TestMockCollectorDefaults(t *testing.T) {
	m := NewMockCollector("test", 5*time.Second)

	if m.Name() != "test" {
		t.Errorf("Name = %q, want %q", m.Name(), "test")
	}
	if m.Interval() != 5*time.Second {
		t.Errorf("Interval = %v, want %v", m.Interval(), 5*time.Second)
	}
	if !m.Healthy() {
		t.Error("default Healthy should be true")
	}
	if m.CallCount() != 0 {
		t.Errorf("initial CallCount = %d, want 0", m.CallCount())
	}
}

func TestMockCollectorWithOptions(t *testing.T) {
	testErr := errors.New("fail")
	m := NewMockCollector("opts", time.Second,
		WithData("hello"),
		WithError(testErr),
		WithHealthy(false),
	)

	if m.Healthy() {
		t.Error("Healthy should be false")
	}

	data, err := m.Collect(context.Background())
	if data != "hello" {
		t.Errorf("Data = %v, want %q", data, "hello")
	}
	if !errors.Is(err, testErr) {
		t.Errorf("Error = %v, want %v", err, testErr)
	}
	if m.CallCount() != 1 {
		t.Errorf("CallCount = %d, want 1", m.CallCount())
	}
}

func TestMockCollectorCallCount(t *testing.T) {
	m := NewMockCollector("counter", time.Second, WithData(42))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		m.Collect(ctx)
	}

	if m.CallCount() != 5 {
		t.Errorf("CallCount = %d, want 5", m.CallCount())
	}
}

func TestMockCollectorWithCollectFunc(t *testing.T) {
	calls := 0
	m := NewMockCollector("custom", time.Second,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			calls++
			return fmt.Sprintf("call-%d", calls), nil
		}),
	)

	data, err := m.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != "call-1" {
		t.Errorf("Data = %v, want %q", data, "call-1")
	}

	data, _ = m.Collect(context.Background())
	if data != "call-2" {
		t.Errorf("Data = %v, want %q", data, "call-2")
	}
}

func TestMockCollectorSetters(t *testing.T) {
	m := NewMockCollector("mut", time.Second)

	m.SetData("updated")
	m.SetError(errors.New("boom"))
	m.SetHealthy(false)

	if m.Healthy() {
		t.Error("Healthy should be false after SetHealthy(false)")
	}

	data, err := m.Collect(context.Background())
	if data != "updated" {
		t.Errorf("Data = %v, want %q", data, "updated")
	}
	if err == nil || err.Error() != "boom" {
		t.Errorf("Error = %v, want 'boom'", err)
	}
}

// --- Runner Tests ---

func TestRunnerReceivesUpdates(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("fast", 50*time.Millisecond, WithData("ping")))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer runner.Stop()

	// Wait for at least one update.
	select {
	case u := <-updates:
		if u.Source != "fast" {
			t.Errorf("Source = %q, want %q", u.Source, "fast")
		}
		if u.Data != "ping" {
			t.Errorf("Data = %v, want %q", u.Data, "ping")
		}
		if u.Error != nil {
			t.Errorf("unexpected error: %v", u.Error)
		}
		if u.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for update")
	}
}

func TestRunnerMultipleCollectors(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("alpha", 50*time.Millisecond, WithData("a")))
	_ = r.Register(NewMockCollector("beta", 50*time.Millisecond, WithData("b")))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = runner.Start(ctx)
	defer runner.Stop()

	seen := make(map[string]bool)
	deadline := time.After(400 * time.Millisecond)

	for len(seen) < 2 {
		select {
		case u := <-updates:
			seen[u.Source] = true
		case <-deadline:
			t.Fatalf("timed out; only saw sources: %v", seen)
		}
	}

	if !seen["alpha"] || !seen["beta"] {
		t.Errorf("expected both alpha and beta, got %v", seen)
	}
}

func TestRunnerGracefulDegradation(t *testing.T) {
	r := NewRegistry()
	testErr := errors.New("broken")
	_ = r.Register(NewMockCollector("failing", 50*time.Millisecond, WithError(testErr)))
	_ = r.Register(NewMockCollector("working", 50*time.Millisecond, WithData("ok")))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = runner.Start(ctx)
	defer runner.Stop()

	var sawFailing, sawWorking bool
	deadline := time.After(400 * time.Millisecond)

	for !sawFailing || !sawWorking {
		select {
		case u := <-updates:
			switch u.Source {
			case "failing":
				sawFailing = true
				if u.Error == nil {
					t.Error("failing collector should report error")
				}
			case "working":
				sawWorking = true
				if u.Error != nil {
					t.Errorf("working collector had error: %v", u.Error)
				}
			}
		case <-deadline:
			t.Fatalf("timed out; sawFailing=%v sawWorking=%v", sawFailing, sawWorking)
		}
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	r := NewRegistry()

	blocked := make(chan struct{})
	_ = r.Register(NewMockCollector("slow", 50*time.Millisecond,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case blocked <- struct{}{}:
				return "done", nil
			}
		}),
	))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithCancel(context.Background())
	_ = runner.Start(ctx)

	// Wait for at least one collection to be attempted.
	select {
	case <-blocked:
	case <-time.After(2 * time.Second):
		t.Fatal("collector never ran")
	}

	cancel()

	// Runner should stop quickly.
	done := make(chan struct{})
	go func() {
		runner.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good.
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return after context cancellation")
	}
}

func TestRunnerStopWaitsForGoroutines(t *testing.T) {
	r := NewRegistry()

	var collectCount int64
	var mu sync.Mutex

	_ = r.Register(NewMockCollector("tracked", 30*time.Millisecond,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			mu.Lock()
			collectCount++
			mu.Unlock()
			return nil, nil
		}),
	))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithCancel(context.Background())
	_ = runner.Start(ctx)

	// Let some collections happen.
	time.Sleep(150 * time.Millisecond)
	cancel()
	runner.Stop()

	mu.Lock()
	count := collectCount
	mu.Unlock()

	// After Stop, no more collections should occur.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	countAfter := collectCount
	mu.Unlock()

	if countAfter != count {
		t.Errorf("collections continued after Stop: before=%d, after=%d", count, countAfter)
	}
}

func TestRunnerRunOnce(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("manual", time.Hour, WithData("triggered")))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	data, err := runner.RunOnce(context.Background(), "manual")
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	if data != "triggered" {
		t.Errorf("Data = %v, want %q", data, "triggered")
	}

	// Status should be updated.
	s, ok := r.Status("manual")
	if !ok {
		t.Fatal("Status not found after RunOnce")
	}
	if s.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", s.RunCount)
	}
	if s.LastRun.IsZero() {
		t.Error("LastRun should not be zero after RunOnce")
	}
	if s.LastLatency <= 0 {
		t.Error("LastLatency should be positive")
	}
}

func TestRunnerRunOnceNotFound(t *testing.T) {
	r := NewRegistry()
	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	_, err := runner.RunOnce(context.Background(), "ghost")
	if err == nil {
		t.Fatal("RunOnce should error for unregistered collector")
	}
}

func TestRunnerRunOnceWithError(t *testing.T) {
	r := NewRegistry()
	testErr := errors.New("runonce-fail")
	_ = r.Register(NewMockCollector("errorer", time.Hour, WithError(testErr)))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	_, err := runner.RunOnce(context.Background(), "errorer")
	if !errors.Is(err, testErr) {
		t.Fatalf("RunOnce error = %v, want %v", err, testErr)
	}

	s, _ := r.Status("errorer")
	if s.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", s.ErrorCount)
	}
	if s.Healthy {
		t.Error("status should be unhealthy after error")
	}
}

func TestRunnerHealth(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("good", time.Hour, WithData("ok")))
	_ = r.Register(NewMockCollector("bad", time.Hour, WithError(errors.New("fail"))))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	// Initially all healthy.
	health := runner.Health()
	if !health["good"] || !health["bad"] {
		t.Errorf("initial health should all be true: %v", health)
	}

	// Trigger bad collector to mark it unhealthy.
	runner.RunOnce(context.Background(), "bad")

	health = runner.Health()
	if !health["good"] {
		t.Error("good should still be healthy")
	}
	if health["bad"] {
		t.Error("bad should be unhealthy after error")
	}
}

func TestRunnerHealthEmpty(t *testing.T) {
	r := NewRegistry()
	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	health := runner.Health()
	if len(health) != 0 {
		t.Errorf("Health should be empty for empty registry, got %v", health)
	}
}

func TestRunnerStatusTracking(t *testing.T) {
	r := NewRegistry()
	callNum := 0
	testErr := errors.New("intermittent")

	_ = r.Register(NewMockCollector("tracked", 30*time.Millisecond,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			callNum++
			if callNum%2 == 0 {
				return nil, testErr
			}
			return "ok", nil
		}),
	))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_ = runner.Start(ctx)

	// Wait for several cycles.
	time.Sleep(250 * time.Millisecond)
	runner.Stop()

	s, ok := r.Status("tracked")
	if !ok {
		t.Fatal("Status not found")
	}
	if s.RunCount < 3 {
		t.Errorf("RunCount = %d, want >= 3", s.RunCount)
	}
	if s.ErrorCount == 0 {
		t.Error("ErrorCount should be > 0 for intermittent failures")
	}
	if s.LastLatency <= 0 {
		t.Error("LastLatency should be positive")
	}
}

func TestRunnerEmptyRegistry(t *testing.T) {
	r := NewRegistry()
	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	if err := runner.Start(context.Background()); err != nil {
		t.Fatalf("Start with empty registry should not error: %v", err)
	}

	// Stop should not block.
	done := make(chan struct{})
	go func() {
		runner.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop blocked on empty registry")
	}
}

func TestRunnerDifferentIntervals(t *testing.T) {
	r := NewRegistry()

	fastCalls := &callCounter{}
	slowCalls := &callCounter{}

	_ = r.Register(NewMockCollector("fast", 20*time.Millisecond,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			fastCalls.inc()
			return nil, nil
		}),
	))
	_ = r.Register(NewMockCollector("slow", 100*time.Millisecond,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			slowCalls.inc()
			return nil, nil
		}),
	))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_ = runner.Start(ctx)
	time.Sleep(250 * time.Millisecond)
	runner.Stop()

	fc := fastCalls.get()
	sc := slowCalls.get()

	// Fast should have significantly more calls than slow.
	if fc <= sc {
		t.Errorf("fast calls (%d) should exceed slow calls (%d)", fc, sc)
	}
}

func TestRunnerImmediateCollection(t *testing.T) {
	// Verify that a collector runs immediately on Start, not after the first
	// interval tick.
	r := NewRegistry()

	collected := make(chan struct{}, 1)
	_ = r.Register(NewMockCollector("immediate", time.Hour,
		WithCollectFunc(func(ctx context.Context) (interface{}, error) {
			select {
			case collected <- struct{}{}:
			default:
			}
			return "first", nil
		}),
	))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = runner.Start(ctx)
	defer runner.Stop()

	select {
	case <-collected:
		// Good - collected immediately.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("collector should run immediately on Start, not wait for first tick")
	}
}

func TestRunnerConcurrentRegistrySafety(t *testing.T) {
	r := NewRegistry()

	// Register collectors concurrently.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent-%d", n)
			_ = r.Register(NewMockCollector(name, time.Second))
		}(i)
	}
	wg.Wait()

	names := r.List()
	if len(names) != 10 {
		t.Errorf("expected 10 collectors, got %d", len(names))
	}

	// Read statuses concurrently while they exist.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.AllStatus()
			_ = r.List()
		}()
	}
	wg.Wait()
}

func TestUpdateFields(t *testing.T) {
	now := time.Now()
	testErr := errors.New("test")

	u := Update{
		Source:    "test-source",
		Data:     map[string]int{"cpu": 42},
		Timestamp: now,
		Error:    testErr,
	}

	if u.Source != "test-source" {
		t.Errorf("Source = %q, want %q", u.Source, "test-source")
	}
	data, ok := u.Data.(map[string]int)
	if !ok {
		t.Fatal("Data type assertion failed")
	}
	if data["cpu"] != 42 {
		t.Errorf("Data[cpu] = %d, want 42", data["cpu"])
	}
	if !u.Timestamp.Equal(now) {
		t.Errorf("Timestamp mismatch")
	}
	if !errors.Is(u.Error, testErr) {
		t.Errorf("Error = %v, want %v", u.Error, testErr)
	}
}

func TestCollectorStatusZeroValue(t *testing.T) {
	var s CollectorStatus
	if s.Name != "" {
		t.Errorf("zero Name should be empty, got %q", s.Name)
	}
	if s.Healthy {
		t.Error("zero Healthy should be false")
	}
	if !s.LastRun.IsZero() {
		t.Error("zero LastRun should be zero time")
	}
	if s.RunCount != 0 {
		t.Errorf("zero RunCount should be 0, got %d", s.RunCount)
	}
}

func TestRunnerStopIdempotent(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("x", 50*time.Millisecond, WithData(1)))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	ctx, cancel := context.WithCancel(context.Background())
	_ = runner.Start(ctx)
	cancel()

	// Calling Stop multiple times should not panic.
	runner.Stop()
	runner.Stop()
	runner.Stop()
}

func TestRunnerRunOnceUpdatesStatus(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(NewMockCollector("statuscheck", time.Hour, WithData("val")))

	updates := make(chan Update, DefaultUpdateBufferSize)
	runner := NewRunner(r, updates)

	// Run twice.
	runner.RunOnce(context.Background(), "statuscheck")
	runner.RunOnce(context.Background(), "statuscheck")

	s, _ := r.Status("statuscheck")
	if s.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2", s.RunCount)
	}
	if !s.Healthy {
		t.Error("should be healthy after successful runs")
	}
}

// --- helpers ---

type callCounter struct {
	mu    sync.Mutex
	count int64
}

func (c *callCounter) inc() {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}

func (c *callCounter) get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}
