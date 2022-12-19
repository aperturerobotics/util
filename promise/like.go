package promise

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// PromiseLike is any object which satisfies the Promise interface.
type PromiseLike[T any] interface {
	// SetResult sets the result of the promise.
	//
	// Returns false if the result was already set.
	SetResult(val T, err error) bool
	// Await awaits for a result or for ctx to be canceled.
	Await(ctx context.Context) (val T, err error)
	// AwaitWithErrCh waits for the result to be set or for an error to be pushed to the channel.
	AwaitWithErrCh(ctx context.Context, errCh <-chan error) (val T, err error)
}

// CheckPromiseLike runs some tests against the PromiseLike.
//
// intended to be used in go tests
func CheckPromiseLike(ctx context.Context, ctor func() PromiseLike[int]) error {
	p1 := ctor()

	// test context canceled during await
	p1Ctx, p1CtxCancel := context.WithCancel(ctx)
	go func() {
		<-time.After(time.Millisecond * 50)
		p1CtxCancel()
	}()
	_, err := p1.Await(p1Ctx)
	if err != context.Canceled {
		return errors.New("expected await to return context canceled")
	}

	// test SetResult during Await
	go func() {
		<-time.After(time.Millisecond * 50)
		_ = p1.SetResult(5, nil)
	}()
	val, err := p1.Await(ctx)
	if err != nil {
		return err
	}
	if val != 5 {
		return errors.Errorf("expected value 5 but got %v", val)
	}

	return nil
}
