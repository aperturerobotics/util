package broadcast

import (
	"sync"
)

// Broadcast implements notifying waiters via a channel.
//
// The zero-value of this struct is valid.
type Broadcast struct {
	mtx sync.Mutex
	ch  *broadcastWaitCh
}

type broadcastWaitCh struct {
	once sync.Once
	ch   chan struct{}
}

func newBroadcastWaitCh() *broadcastWaitCh {
	return &broadcastWaitCh{ch: make(chan struct{})}
}

func (c *broadcastWaitCh) close() {
	c.once.Do(func() {
		close(c.ch)
	})
}
