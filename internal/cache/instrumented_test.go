package cache

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// getCounterVecValue reads the current value of a CounterVec for the given label.
func getCounterVecValue(cv *prometheus.CounterVec, label string) float64 {
	c, err := cv.GetMetricWithLabelValues(label)
	if err != nil {
		return 0
	}
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

// newInstrumentedTestCache creates an instrumented memory cache with the given group and
// registers a cleanup that calls Close() at the end of the test.
func newInstrumentedTestCache(t *testing.T, group string) Cache {
	t.Helper()
	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour, Group: group})
	if err != nil {
		t.Fatalf("New instrumented cache: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestInstrumentedCache_Hits(t *testing.T) {
	c := newInstrumentedTestCache(t, "test-hits")

	c.Set("k", []byte("v"))
	before := getCounterVecValue(HitsTotal, "test-hits")

	_, _ = c.Get("k") // hit

	after := getCounterVecValue(HitsTotal, "test-hits")
	if after != before+1 {
		t.Errorf("Expected hits to increment by 1, got diff %.0f", after-before)
	}
}

func TestInstrumentedCache_Misses(t *testing.T) {
	c := newInstrumentedTestCache(t, "test-misses")

	before := getCounterVecValue(MissesTotal, "test-misses")

	_, _ = c.Get("absent") // miss

	after := getCounterVecValue(MissesTotal, "test-misses")
	if after != before+1 {
		t.Errorf("Expected misses to increment by 1, got diff %.0f", after-before)
	}
}

func TestInstrumentedCache_Evictions(t *testing.T) {
	evicted := make([]string, 0)
	onEvict := func(key string, _ []byte) {
		evicted = append(evicted, key)
	}

	// Size=2 so the third Set triggers an eviction.
	c, err := New("memory", ProviderConfig{Size: 2, TTL: time.Hour, Group: "test-evict", OnEvict: onEvict})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	before := getCounterVecValue(EvictionsTotal, "test-evict")

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3")) // evicts "a"

	after := getCounterVecValue(EvictionsTotal, "test-evict")
	if after != before+1 {
		t.Errorf("Expected evictions to increment by 1, got diff %.0f", after-before)
	}

	// Original OnEvict callback must still fire.
	if len(evicted) != 1 || evicted[0] != "a" {
		t.Errorf("Expected original OnEvict to fire for key 'a', got %v", evicted)
	}
}

func TestInstrumentedCache_EntriesLazy(t *testing.T) {
	// Use an isolated registry so we can gather only the entries we care about.
	reg := prometheus.NewRegistry()

	origReg := entriesReg
	entriesReg = reg
	t.Cleanup(func() { entriesReg = origReg })

	c := newInstrumentedTestCache(t, "test-entries")

	// Helper: gather the cache_entries gauge for our group from reg.
	gatherEntries := func() float64 {
		mfs, _ := reg.Gather()
		for _, mf := range mfs {
			if mf.GetName() != "cache_entries" {
				continue
			}
			for _, m := range mf.GetMetric() {
				for _, lp := range m.GetLabel() {
					if lp.GetName() == "cache" && lp.GetValue() == "test-entries" {
						return m.GetGauge().GetValue()
					}
				}
			}
		}
		return -1
	}

	if v := gatherEntries(); v != 0 {
		t.Fatalf("Expected 0 entries before Set, got %.0f", v)
	}

	c.Set("x", []byte("1"))
	c.Set("y", []byte("2"))

	// Len() is queried at scrape time, so the gauge reflects the real count.
	if v := gatherEntries(); v != 2 {
		t.Errorf("Expected 2 entries after two Sets, got %.0f", v)
	}
}

func TestInstrumentedCache_Close_UnregistersEntries(t *testing.T) {
	reg := prometheus.NewRegistry()

	origReg := entriesReg
	entriesReg = reg
	t.Cleanup(func() { entriesReg = origReg })

	c, err := New("memory", ProviderConfig{Size: 10, TTL: time.Hour, Group: "test-close"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Collector must be registered after creation.
	entriesCollectorMu.Lock()
	_, registered := entriesCollectors["test-close"]
	entriesCollectorMu.Unlock()
	if !registered {
		t.Fatal("Expected entries collector to be registered after New()")
	}

	_ = c.Close()

	// Collector must be gone after Close().
	entriesCollectorMu.Lock()
	_, registered = entriesCollectors["test-close"]
	entriesCollectorMu.Unlock()
	if registered {
		t.Fatal("Expected entries collector to be unregistered after Close()")
	}
}
