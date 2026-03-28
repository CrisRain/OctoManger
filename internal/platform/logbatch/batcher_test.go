package logbatch

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBatcherFlushesOnMaxBatch(t *testing.T) {
	flushCh := make(chan []int, 1)
	b := New(func(ctx context.Context, items []int) error {
		cpy := append([]int(nil), items...)
		flushCh <- cpy
		return nil
	})
	b.maxBatch = 2
	b.interval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	go b.Run(ctx)

	b.Add(1)
	b.Add(2)

	select {
	case got := <-flushCh:
		if len(got) != 2 || got[0] != 1 || got[1] != 2 {
			t.Fatalf("unexpected batch %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for flush")
	}

	cancel()
	b.Wait()
}

func TestBatcherFlushesOnCancel(t *testing.T) {
	flushCh := make(chan []int, 1)
	b := New(func(ctx context.Context, items []int) error {
		cpy := append([]int(nil), items...)
		flushCh <- cpy
		return nil
	})
	b.maxBatch = 10
	b.interval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	go b.Run(ctx)

	b.Add(42)
	cancel()
	b.Wait()

	select {
	case got := <-flushCh:
		if len(got) != 1 || got[0] != 42 {
			t.Fatalf("unexpected batch %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for cancel flush")
	}
}

func TestBatcherErrorHandler(t *testing.T) {
	errCh := make(chan error, 1)
	b := New(func(ctx context.Context, items []int) error {
		return errors.New("flush failed")
	}).WithErrorHandler(func(err error) {
		errCh <- err
	})
	b.maxBatch = 1
	b.interval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	go b.Run(ctx)
	b.Add(7)

	select {
	case err := <-errCh:
		if err == nil || err.Error() != "flush failed" {
			t.Fatalf("unexpected error %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for error handler")
	}

	cancel()
	b.Wait()
}
