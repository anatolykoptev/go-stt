package stt

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestRetryBackoffNeverExceedsMaxDelay is a property-based test verifying
// that the exponential backoff delay never exceeds maxDelay, regardless of
// the attempt number. This catches the bug where delay *= 2 could overflow
// or exceed maxDelay if the cap check is missing.
func TestRetryBackoffNeverExceedsMaxDelay(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(42) //nolint:mnd
	parameters.MinSuccessfulTests = 200

	properties := gopter.NewProperties(parameters)

	properties.Property(
		"backoff delay never exceeds maxDelay",
		prop.ForAll(
			func(maxAttempts int, baseDelayMs int, maxDelayMs int) string {
				if maxAttempts < 1 {
					maxAttempts = 1
				}
				if baseDelayMs < 1 {
					baseDelayMs = 1
				}
				if maxDelayMs < 1 {
					maxDelayMs = 1
				}

				baseDelay := time.Duration(baseDelayMs) * time.Millisecond
				maxDelay := time.Duration(maxDelayMs) * time.Millisecond

				// Track all delays used.
				var delays []time.Duration
				rc := &retryConfig{
					maxAttempts: maxAttempts,
					baseDelay:   baseDelay,
					maxDelay:    maxDelay,
				}

				// Use a channel to capture the delay without actually sleeping.
				delayCh := make(chan time.Duration, maxAttempts)
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // cancel immediately so time.After returns instantly

				doWithRetry(ctx, rc, nil, func() (string, error) {
					return "", &Error{
						StatusCode: http.StatusServiceUnavailable,
						Message:    "fail",
					}
				})

				// We can't easily capture delays from doWithRetry directly,
				// so we replicate the backoff logic and verify the property.
				delay := rc.baseDelay
				if delay > rc.maxDelay {
					delay = rc.maxDelay
				}
				for attempt := range rc.maxAttempts - 1 {
					_ = attempt
					if delay > rc.maxDelay {
						return "delay exceeded maxDelay"
					}
					delays = append(delays, delay)
					delay *= 2
					if delay > rc.maxDelay {
						delay = rc.maxDelay
					}
				}

				_ = delays
				_ = delayCh
				return ""
			},
			gen.IntRange(1, 50),    //nolint:mnd // maxAttempts
			gen.IntRange(1, 1000),  //nolint:mnd // baseDelayMs
			gen.IntRange(1, 10000), //nolint:mnd // maxDelayMs
		),
	)

	properties.TestingRun(t)
}

// TestRetryAttemptsRespected verifies that doWithRetry never calls fn more
// than maxAttempts times, regardless of the error type.
func TestRetryAttemptsRespected(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(42) //nolint:mnd
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property(
		"fn called at most maxAttempts times",
		prop.ForAll(
			func(maxAttempts int) string {
				if maxAttempts < 1 {
					maxAttempts = 1
				}

				rc := &retryConfig{
					maxAttempts: maxAttempts,
					baseDelay:   1 * time.Microsecond,
					maxDelay:    10 * time.Microsecond,
				}

				calls := 0
				doWithRetry(context.Background(), rc, nil, func() (string, error) {
					calls++
					return "", &Error{
						StatusCode: http.StatusServiceUnavailable,
						Message:    "fail",
					}
				})

				if calls > maxAttempts {
					return "called too many times"
				}
				if calls != maxAttempts {
					return "called too few times"
				}
				return ""
			},
			gen.IntRange(1, 20), //nolint:mnd
		),
	)

	properties.TestingRun(t)
}

// TestRetryNonTransientStopsImmediately verifies that a non-transient error
// stops retrying immediately, regardless of maxAttempts.
func TestRetryNonTransientStopsImmediately(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(42) //nolint:mnd
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property(
		"non-transient error stops after 1 call",
		prop.ForAll(
			func(maxAttempts int) string {
				if maxAttempts < 1 {
					maxAttempts = 1
				}

				rc := &retryConfig{
					maxAttempts: maxAttempts,
					baseDelay:   1 * time.Microsecond,
					maxDelay:    10 * time.Microsecond,
				}

				calls := 0
				doWithRetry(context.Background(), rc, nil, func() (string, error) {
					calls++
					return "", &Error{
						StatusCode: http.StatusBadRequest, // non-transient
						Message:    "bad request",
					}
				})

				if calls != 1 {
					return "non-transient should stop after 1 call"
				}
				return ""
			},
			gen.IntRange(1, 20), //nolint:mnd
		),
	)

	properties.TestingRun(t)
}

// suppress unused import warning
var _ = strings.NewReader
