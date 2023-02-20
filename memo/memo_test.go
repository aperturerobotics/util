package memo

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestMemoizeFunc tests memoizing a function.
func TestMemoizeFunc(t *testing.T) {
	var n int
	complete := make(chan struct{})
	fn := func() (int, error) {
		n++
		<-complete
		return n, nil
	}
	memoFn := MemoizeFunc(fn)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = memoFn()
		}()
	}
	var returned atomic.Bool
	go func() {
		_, _ = memoFn()
		returned.Store(true)
	}()
	<-time.After(time.Millisecond * 50)
	if returned.Load() {
		t.Fail()
	}
	close(complete)
	<-time.After(time.Millisecond * 50)
	res, err := memoFn()
	if err != nil || res != 1 {
		t.Fail()
	}
	if !returned.Load() {
		t.Fail()
	}
}
