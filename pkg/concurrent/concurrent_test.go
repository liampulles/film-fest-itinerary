package concurrent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestMapPreservesOrder(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}

	got, err := Map(context.Background(), inputs, func(_ context.Context, n int) (int, error) {
		return n * 2, nil
	}, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{2, 4, 6, 8, 10}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("result[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestMapEmptyInput(t *testing.T) {
	got, err := Map(context.Background(), []int{}, func(_ context.Context, n int) (int, error) {
		return n, nil
	}, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestMapRetriesThenSucceeds(t *testing.T) {
	var calls atomic.Int32

	got, err := Map(context.Background(), []int{1}, func(_ context.Context, n int) (int, error) {
		// Fail twice, succeed on the third (final) attempt.
		if calls.Add(1) < defaultMaxRetries {
			return 0, errors.New("transient")
		}
		return n, nil
	}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0] != 1 {
		t.Errorf("result = %d, want 1", got[0])
	}
	if calls.Load() != defaultMaxRetries {
		t.Errorf("attempts = %d, want %d", calls.Load(), defaultMaxRetries)
	}
}

func TestMapCancelsOnContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Map(ctx, []int{1, 2, 3}, func(ctx context.Context, n int) (int, error) {
		// Block until the context is cancelled.
		<-ctx.Done()
		return 0, ctx.Err()
	}, 3)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}
