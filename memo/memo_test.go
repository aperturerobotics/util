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
		go memoFn()
	}
	var returned atomic.Bool
	go func() {
		memoFn()
		returned.Store(true)
	}()
	<-time.After(time.Millisecond * 50)
	if returned.Load() {
		t.Fail()
	}
	close(complete)
	res, err := memoFn()
	if err != nil || res != 1 {
		t.Fail()
	}
	if !returned.Load() {
		t.Fail()
	}
}
