package broadcast

import "testing"

func TestBroadcast(t *testing.T) {
	var bcast Broadcast

	ch := bcast.GetWaitCh()
	select {
	case <-ch:
		t.FailNow()
	default:
	}

	bcast.Broadcast()
	select {
	case <-ch:
	default:
		t.FailNow()
	}
	select {
	case <-bcast.GetWaitCh():
		t.FailNow()
	default:
	}
}
