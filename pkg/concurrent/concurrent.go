// Package concurrent provides generic helpers for running work concurrently
// across a bounded pool of workers.
package concurrent

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// defaultMaxRetries is how many times Map will retry the mapping function
	// for a single input before giving up.
	defaultMaxRetries = 3
	// defaultBaseDelay is the backoff before the first retry; it doubles on
	// each subsequent retry.
	defaultBaseDelay = 100 * time.Millisecond
)

// Map applies fn to every element of inputs concurrently, using at most
// workers goroutines, and returns the results in the same order as inputs.
//
// fn is retried with exponential backoff on error (see defaultMaxRetries and
// defaultBaseDelay). If an input still fails after all attempts, Map cancels
// the remaining work and returns that error. Map also stops early if ctx is
// cancelled or its deadline passes, returning ctx's error.
//
// Map is agnostic to what fn does — callers thread the provided context into
// their own work (e.g. HTTP requests) so cancellation propagates.
func Map[In, Out any](
	ctx context.Context,
	inputs []In,
	fn func(context.Context, In) (Out, error),
	workers int,
) ([]Out, error) {
	results := make([]Out, len(inputs))
	if len(inputs) == 0 {
		return results, nil
	}

	if workers < 1 {
		workers = 1
	}
	if workers > len(inputs) {
		workers = len(inputs)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	for i, in := range inputs {
		g.Go(func() error {
			// Don't start new work once the group is cancelled.
			if err := ctx.Err(); err != nil {
				return err
			}

			out, err := withRetry(ctx, func(ctx context.Context) (Out, error) {
				return fn(ctx, in)
			}, 0)
			if err != nil {
				return err
			}

			results[i] = out
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// withRetry runs fn, retrying with exponential backoff until it succeeds, the
// attempt budget is exhausted, or ctx is done. attempt is zero-based: callers
// start it at 0.
func withRetry[Out any](ctx context.Context, fn func(context.Context) (Out, error), attempt int) (Out, error) {
	if attempt > defaultMaxRetries {
		var zero Out
		return zero, errors.New("retries exceeded")
	}

	// Sleep for a bit, or break if the context is cancelled. At the start this will return immediately.
	delay := defaultBaseDelay * time.Duration(attempt*2)
	if waitErr := cancelOrSleep(ctx, delay); waitErr != nil {
		var zero Out
		return zero, waitErr
	}

	// Try
	out, err := fn(ctx)
	if err != nil {
		// Retry later
		return withRetry(ctx, fn, attempt+1)
	}

	// All good, done.
	return out, nil
}

// Potentially break if the context is cancelled, else sleep.
func cancelOrSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
