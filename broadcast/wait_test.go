package broadcast

import (
	"context"
	"testing"
	"time"
)

func TestWaitAny_IgnoresNilChannels(t *testing.T) {
	ch := make(chan struct{})
	close(ch)

	if err := WaitAny(context.Background(), nil, ch, nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestWaitAny_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- WaitAny(ctx, nil)
	}()

	select {
	case err := <-done:
		t.Fatalf("WaitAny returned before cancellation: %v", err)
	case <-time.After(10 * time.Millisecond):
	}

	cancel()

	if err := <-done; err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWaitAny_FirstWake(t *testing.T) {
	ctx := t.Context()

	ch := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WaitAny(ctx, ch)
	}()

	select {
	case err := <-done:
		t.Fatalf("WaitAny returned before wake: %v", err)
	case <-time.After(10 * time.Millisecond):
	}

	close(ch)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitAny did not return after wake")
	}
}

func TestWaitAny_RepeatedWake(t *testing.T) {
	ctx := context.Background()
	for range 3 {
		ch := make(chan struct{})
		close(ch)
		if err := WaitAny(ctx, ch); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	}
}

func TestWaitAny_NoIgnoredSources(t *testing.T) {
	for sourceIdx := range 5 {
		ctx, cancel := context.WithCancel(context.Background())

		chans := make([]chan struct{}, 5)
		waitChs := make([]<-chan struct{}, 0, len(chans)+2)
		waitChs = append(waitChs, nil)
		for i := range chans {
			chans[i] = make(chan struct{})
			waitChs = append(waitChs, chans[i])
		}
		waitChs = append(waitChs, nil)

		done := make(chan error, 1)
		go func() {
			done <- WaitAny(ctx, waitChs...)
		}()

		select {
		case err := <-done:
			cancel()
			t.Fatalf("WaitAny returned before source %d woke: %v", sourceIdx, err)
		case <-time.After(10 * time.Millisecond):
		}

		close(chans[sourceIdx])

		select {
		case err := <-done:
			cancel()
			if err != nil {
				t.Fatalf("expected nil error for source %d, got %v", sourceIdx, err)
			}
		case <-time.After(time.Second):
			cancel()
			t.Fatalf("WaitAny ignored source %d", sourceIdx)
		}
	}
}
