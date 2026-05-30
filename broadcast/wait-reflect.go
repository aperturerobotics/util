//go:build !tinygo

package broadcast

import (
	"context"
	"reflect"
)

func waitAny(ctx context.Context, waitChs []<-chan struct{}) error {
	cases := make([]reflect.SelectCase, 0, len(waitChs)+1)
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})
	for _, ch := range waitChs {
		if ch == nil {
			continue
		}
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}

	if len(cases) == 1 {
		<-ctx.Done()
		return ctx.Err()
	}

	chosen, _, _ := reflect.Select(cases)
	if chosen == 0 {
		return ctx.Err()
	}
	return nil
}
