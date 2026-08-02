package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/config"
	"github.com/Belphemur/SuperSubtitles/v2/internal/metrics"
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/rs/zerolog"
)

// Default circuit breaker thresholds, used whenever the corresponding config
// field is left unset (zero value / empty string).
const (
	defaultCircuitBreakerFailureThreshold uint          = 5
	defaultCircuitBreakerSuccessThreshold uint          = 2
	defaultCircuitBreakerOpenDuration     time.Duration = 30 * time.Second
)

// newCircuitBreakerPolicy builds a failsafe-go circuit breaker that protects
// every HTTP call made to feliratok.eu. It opens after a run of consecutive
// failures (any 5xx response including non-standard codes such as 530, 429 Too
// Many Requests, or connection errors), short-circuiting further requests with
// circuitbreaker.ErrOpen so the upstream site gets a chance to recover instead
// of being hammered with requests that are very likely to fail. After the
// configured open duration it transitions to half-open and allows a small
// number of trial requests through before fully closing again.
func newCircuitBreakerPolicy(cfg *config.Config, logger zerolog.Logger) failsafe.Policy[*http.Response] {
	failureThreshold := cfg.CircuitBreaker.FailureThreshold
	if failureThreshold == 0 {
		failureThreshold = defaultCircuitBreakerFailureThreshold
	}

	successThreshold := cfg.CircuitBreaker.SuccessThreshold
	if successThreshold == 0 {
		successThreshold = defaultCircuitBreakerSuccessThreshold
	}

	openDuration := defaultCircuitBreakerOpenDuration
	if cfg.CircuitBreaker.OpenDuration != "" {
		if parsed, err := time.ParseDuration(cfg.CircuitBreaker.OpenDuration); err != nil {
			logger.Warn().Err(err).Str("open_duration", cfg.CircuitBreaker.OpenDuration).Msg("Invalid circuit breaker open duration, using default 30s")
		} else {
			openDuration = parsed
		}
	}

	builder := circuitbreaker.NewBuilder[*http.Response]().
		HandleIf(isTransientHTTPFailure).
		WithFailureThreshold(failureThreshold).
		WithSuccessThreshold(successThreshold).
		WithDelay(openDuration).
		OnOpen(func(e circuitbreaker.StateChangedEvent) {
			metrics.CircuitBreakerState.Set(2)
			logger.Error().
				Str("previous_state", e.OldState.String()).
				Str("state", "open").
				Dur("retry_after", openDuration).
				Msg("Circuit breaker opened: feliratok.eu is failing repeatedly, short-circuiting further requests")
		}).
		OnHalfOpen(func(e circuitbreaker.StateChangedEvent) {
			metrics.CircuitBreakerState.Set(1)
			logger.Warn().
				Str("previous_state", e.OldState.String()).
				Str("state", "half-open").
				Msg("Circuit breaker half-open: allowing trial requests to feliratok.eu")
		}).
		OnClose(func(e circuitbreaker.StateChangedEvent) {
			metrics.CircuitBreakerState.Set(0)
			logger.Info().
				Str("previous_state", e.OldState.String()).
				Str("state", "closed").
				Msg("Circuit breaker closed: feliratok.eu requests are succeeding again")
		})

	return builder.Build()
}

// isTransientHTTPFailure classifies responses that should count as failures for
// the circuit breaker: connection errors, 429 Too Many Requests, and any 5xx
// server error (including non-standard codes such as 530 used by CDNs/proxies).
// Caller-initiated cancellations (context.Canceled) are not upstream faults and
// are excluded so client disconnects do not contribute to opening the breaker.
func isTransientHTTPFailure(resp *http.Response, err error) bool {
	if err != nil {
		// A cancelled caller context is not an upstream fault. Do not let
		// client-side cancellations open the circuit.
		if errors.Is(err, context.Canceled) {
			return false
		}
		return true
	}

	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return true
		}
		if resp.StatusCode >= 500 {
			return true
		}
	}

	return false
}
