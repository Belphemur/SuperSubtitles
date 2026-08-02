package client

// Tests for failsafe-go retry/resilience behaviour in the client package.
// Each test verifies that transient failures (5xx, 429, connection errors) are
// retried the expected number of times before the client either succeeds or
// gives up.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/testutil"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
)

// newTestClientWithRetry creates a client configured with up to maxAttempts total
// attempts and no inter-retry delay, so tests run fast.
func newTestClientWithRetry(serverURL string, maxAttempts int) Client {
	cfg := config.Config{
		SuperSubtitleDomain: serverURL,
		ClientTimeout:       "10s",
	}
	cfg.Retry.MaxAttempts = maxAttempts
	// No InitialDelay → no backoff, retries fire immediately (fast tests)
	return NewClient(&cfg)
}

// newTestClientWithCircuitBreaker creates a client with retries disabled (so
// every failed call counts as exactly one circuit breaker failure) and a
// circuit breaker configured with the given thresholds.
func newTestClientWithCircuitBreaker(serverURL string, failureThreshold, successThreshold uint, openDuration string) Client {
	cfg := config.Config{
		SuperSubtitleDomain: serverURL,
		ClientTimeout:       "10s",
	}
	cfg.Retry.MaxAttempts = 1 // disable retries so each call is a single execution
	cfg.CircuitBreaker.FailureThreshold = failureThreshold
	cfg.CircuitBreaker.SuccessThreshold = successThreshold
	cfg.CircuitBreaker.OpenDuration = openDuration
	return NewClient(&cfg)
}

// TestClient_Retry_SucceedsAfterTransientError verifies that the client retries
// on a 500 response and ultimately succeeds once the server recovers.
func TestClient_Retry_SucceedsAfterTransientError(t *testing.T) {
	t.Parallel()

	// Serve a 500 on the first request, then 200 with valid JSON.
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"film":"2","sorozat":"1"}`))
	}))
	defer server.Close()

	c := newTestClientWithRetry(server.URL, 3)
	result, err := c.CheckForUpdates(context.Background(), 1234)

	if err != nil {
		t.Fatalf("Expected success after retry, got error: %v", err)
	}
	if result.FilmCount != 2 || result.SeriesCount != 1 {
		t.Errorf("Unexpected result: %+v", result)
	}
	if requestCount.Load() != 2 {
		t.Errorf("Expected exactly 2 requests (1 failure + 1 success), got %d", requestCount.Load())
	}
}

// TestClient_Retry_ExhaustsAttemptsAndFails verifies that when every attempt
// returns a 500 the client eventually returns an error after all retries are
// exhausted.
func TestClient_Retry_ExhaustsAttemptsAndFails(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	const maxAttempts = 3
	c := newTestClientWithRetry(server.URL, maxAttempts)
	_, err := c.CheckForUpdates(context.Background(), 1234)

	if err == nil {
		t.Fatal("Expected error after retries exhausted, got nil")
	}
	if requestCount.Load() != maxAttempts {
		t.Errorf("Expected %d total requests, got %d", maxAttempts, requestCount.Load())
	}
}

// TestClient_Retry_429TooManyRequestsIsRetried verifies that 429 responses
// trigger a retry.
func TestClient_Retry_429TooManyRequestsIsRetried(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"film":"0","sorozat":"0"}`))
	}))
	defer server.Close()

	c := newTestClientWithRetry(server.URL, 3)
	result, err := c.CheckForUpdates(context.Background(), 5678)

	if err != nil {
		t.Fatalf("Expected success after retry on 429, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if requestCount.Load() != 2 {
		t.Errorf("Expected 2 requests (1 × 429 + 1 × 200), got %d", requestCount.Load())
	}
}

// TestClient_Retry_404NotFoundIsNotRetried verifies that 404 responses are NOT
// retried (they are not transient errors).
func TestClient_Retry_404NotFoundIsNotRetried(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClientWithRetry(server.URL, 3)
	ctx := context.Background()
	_, err := testutil.CollectSubtitles(ctx, c.StreamSubtitles(ctx, 9999))

	if err == nil {
		t.Fatal("Expected error for 404, got nil")
	}
	// 404 must not be retried — exactly one request should have been made.
	if requestCount.Load() != 1 {
		t.Errorf("Expected exactly 1 request (no retry on 404), got %d", requestCount.Load())
	}
}

// TestClient_Retry_ShowListSucceedsAfterTransientError verifies that show-list
// endpoint calls are retried transparently.
func TestClient_Retry_ShowListSucceedsAfterTransientError(t *testing.T) {
	t.Parallel()

	showHTML := testutil.GenerateShowTableHTML([]testutil.ShowRowOptions{
		{ShowID: 101, ShowName: "Retried Show", Year: 2025},
	})

	// First call for each endpoint path returns 500; subsequent calls return 200.
	// sync.Map is used to safely count per-path requests across concurrent goroutines.
	var requestCounts sync.Map

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.RawQuery
		// LoadOrStore returns the existing *atomic.Int32 if the key is already
		// present, or stores (and returns) a freshly allocated one on first access.
		// Any extra new(atomic.Int32) that loses the race is simply GC'd.
		actual, _ := requestCounts.LoadOrStore(key, new(atomic.Int32))
		cnt := actual.(*atomic.Int32).Add(1)

		if cnt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(showHTML))
	}))
	defer server.Close()

	c := newTestClientWithRetry(server.URL, 3)
	ctx := context.Background()
	shows, err := testutil.CollectShows(ctx, c.StreamShowList(ctx))

	if err != nil {
		t.Fatalf("Expected success after retry, got: %v", err)
	}
	// Each of the 3 endpoints returned 101 — but they all serve the same HTML so
	// deduplication by show ID means only 1 unique show is expected.
	if len(shows) != 1 {
		t.Fatalf("Expected 1 deduplicated show, got %d", len(shows))
	}
	if shows[0].ID != 101 {
		t.Errorf("Expected show ID 101, got %d", shows[0].ID)
	}
}

// TestClient_CircuitBreaker_OpensAfterConsecutiveFailures verifies that once
// the configured number of consecutive failures occurs, the circuit breaker
// opens and short-circuits further requests without hitting the server,
// returning an error that wraps circuitbreaker.ErrOpen.
func TestClient_CircuitBreaker_OpensAfterConsecutiveFailures(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	const failureThreshold = 2
	c := newTestClientWithCircuitBreaker(server.URL, failureThreshold, 1, "1m")
	ctx := context.Background()

	// The first failureThreshold calls should reach the server and fail with a
	// regular (non-circuit-breaker) error.
	for i := 0; i < failureThreshold; i++ {
		_, err := c.CheckForUpdates(ctx, 1)
		if err == nil {
			t.Fatalf("Expected error on attempt %d, got nil", i+1)
		}
		if errors.Is(err, circuitbreaker.ErrOpen) {
			t.Fatalf("Did not expect circuit breaker to be open yet on attempt %d", i+1)
		}
	}
	if requestCount.Load() != failureThreshold {
		t.Fatalf("Expected %d requests before circuit opens, got %d", failureThreshold, requestCount.Load())
	}

	// The next call should be short-circuited: no additional request reaches
	// the server, and the error wraps circuitbreaker.ErrOpen.
	_, err := c.CheckForUpdates(ctx, 1)
	if err == nil {
		t.Fatal("Expected error once circuit breaker is open, got nil")
	}
	if !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("Expected error to wrap circuitbreaker.ErrOpen, got: %v", err)
	}
	if requestCount.Load() != failureThreshold {
		t.Errorf("Expected no additional request while circuit is open, got %d total requests", requestCount.Load())
	}
}

// TestClient_CircuitBreaker_ClosesAfterSuccessInHalfOpen verifies that after
// the open duration elapses, the circuit breaker transitions to half-open and
// closes again once a trial request succeeds.
func TestClient_CircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	t.Parallel()

	var failing atomic.Bool
	failing.Store(true)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if failing.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"film":"0","sorozat":"0"}`))
	}))
	defer server.Close()

	// Open the circuit after a single failure, with a very short open duration
	// so the test can quickly exercise the half-open → closed transition.
	c := newTestClientWithCircuitBreaker(server.URL, 1, 1, "10ms")
	ctx := context.Background()

	// First call fails and opens the circuit.
	if _, err := c.CheckForUpdates(ctx, 1); err == nil {
		t.Fatal("Expected the first call to fail")
	}

	// Immediately retrying should be short-circuited.
	if _, err := c.CheckForUpdates(ctx, 1); !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("Expected circuit breaker open error immediately after opening, got: %v", err)
	}

	// Let the server start succeeding and wait for the open duration to elapse
	// so the breaker transitions to half-open.
	failing.Store(false)
	time.Sleep(50 * time.Millisecond)

	result, err := c.CheckForUpdates(ctx, 1)
	if err != nil {
		t.Fatalf("Expected the half-open trial request to succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// The circuit should now be closed, allowing subsequent calls through.
	if _, err := c.CheckForUpdates(ctx, 1); err != nil {
		t.Fatalf("Expected circuit breaker to be closed after successful trial, got: %v", err)
	}
}
