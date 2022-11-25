package broadcast

import "sync"

// adapted from missinggo (MIT license)
// https://github.com/anacrolix/missinggo/blob/master/chancond.go

// Broadcast implements notifying waiters via a channel.
type Broadcast struct {
	mtx sync.Mutex
	ch  chan struct{}
}

// GetWaitCh returns a channel that is closed when Broadcast is called.
func (c *Broadcast) GetWaitCh() <-chan struct{} {
	c.mtx.Lock()
	if c.ch == nil {
		c.ch = make(chan struct{})
	}
	ch := c.ch
	c.mtx.Unlock()
	return ch
}

func (c *Broadcast) Broadcast() {
	c.mtx.Lock()
	if c.ch != nil {
		close(c.ch)
		c.ch = nil
	}
	c.mtx.Unlock()
}
