package keyed

import (
	"context"
	"sync"
	"time"

	cbackoff "github.com/aperturerobotics/util/backoff/cbackoff"
	"github.com/sirupsen/logrus"
)

// Routine is a function called as a goroutine.
// If nil is returned, exits cleanly permanently.
// If an error is returned, can be restarted later.
type Routine func(ctx context.Context) error

// Keyed manages a set of goroutines with associated Keys.
//
// K is the type of the key.
// V is the type of the value.
type Keyed[K comparable, V any] struct {
	// ctorCb is the constructor callback
	ctorCb func(key K) (Routine, V)
	// exitedCbs is the set of exited callbacks.
	exitedCbs []func(key K, routine Routine, data V, err error)

	// releaseDelay is a delay before stopping a routine.
	releaseDelay time.Duration
	// backoffFactory is the backoff factory
	// if nil, backoff is disabled
	backoffFactory func(k K) cbackoff.BackOff

	// mtx guards below fields
	mtx sync.Mutex
	// ctx is the current root context
	ctx context.Context
	// routines is the set of running routines
	routines map[K]*runningRoutine[K, V]
}

// NewKeyed constructs a new Keyed execution manager.
// Note: routines won't start until SetContext is called.
func NewKeyed[K comparable, V any](
	ctorCb func(key K) (Routine, V),
	opts ...Option[K, V],
) *Keyed[K, V] {
	if ctorCb == nil {
		ctorCb = func(key K) (Routine, V) {
			var empty V
			return nil, empty
		}
	}
	k := &Keyed[K, V]{
		ctorCb: ctorCb,

		routines: make(map[K]*runningRoutine[K, V], 1),
	}
	for _, opt := range opts {
		if opt != nil {
			opt.ApplyToKeyed(k)
		}
	}
	return k
}

// NewKeyedWithLogger constructs a new keyed instance.
// Logs when a controller exits without being removed from the Keys set.
//
// Note: routines won't start until SetContext is called.
func NewKeyedWithLogger[K comparable, V any](
	ctorCb func(key K) (Routine, V),
	le *logrus.Entry,
	opts ...Option[K, V],
) *Keyed[K, V] {
	return NewKeyed(ctorCb, append([]Option[K, V]{WithExitLogger[K, V](le)}, opts...)...)
}

// SetContext updates the root context, restarting all running routines.
//
// nil context is valid and will shutdown the routines.
// if restart is true, all errored routines also restart
func (k *Keyed[K, V]) SetContext(ctx context.Context, restart bool) {
	k.mtx.Lock()
	k.setContextLocked(ctx, restart)
	k.mtx.Unlock()
}

// setContextLocked sets the context while mtx is locked.
func (k *Keyed[K, V]) setContextLocked(ctx context.Context, restart bool) {
	sameCtx := k.ctx == ctx
	if sameCtx && !restart {
		return
	}

	k.ctx = ctx
	for _, rr := range k.routines {
		if sameCtx && rr.err == nil {
			continue
		}
		rr.ctx = nil
		if rr.ctxCancel != nil {
			rr.ctxCancel()
			rr.ctxCancel = nil
		}
		if rr.err == nil || restart {
			if ctx != nil {
				rr.start(ctx, rr.exitedCh, false)
			}
		}
	}
}

// ClearContext clears the context and shuts down any running routines.
func (k *Keyed[K, V]) ClearContext() {
	k.SetContext(nil, false)
}

// GetKeys returns the list of keys registered with the Keyed instance.
func (k *Keyed[K, V]) GetKeys() []K {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	keys := make([]K, 0, len(k.routines))
	for k := range k.routines {
		keys = append(keys, k)
	}
	return keys
}

// KeyWithData is a key with associated data.
type KeyWithData[K comparable, V any] struct {
	// Key is the key.
	Key K
	// Data is the value.
	Data V
}

// GetKeysWithData returns the keys and the data for the keys.
func (k *Keyed[K, V]) GetKeysWithData() []KeyWithData[K, V] {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	out := make([]KeyWithData[K, V], 0, len(k.routines))
	for k, v := range k.routines {
		out = append(out, KeyWithData[K, V]{
			Key:  k,
			Data: v.data,
		})
	}
	return out
}

// SetKey inserts the given key into the set, if it doesn't already exist.
// If start=true, restarts the routine from any stopped or failed state.
// Returns if it existed already or not.
func (k *Keyed[K, V]) SetKey(key K, start bool) (V, bool) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	v, existed := k.routines[key]
	if !existed {
		routine, data := k.ctorCb(key)
		v = newRunningRoutine(k, key, routine, data, k.backoffFactory)
		k.routines[key] = v
	} else {
		if v.deferRemove != nil {
			// cancel removing this key
			_ = v.deferRemove.Stop()
			v.deferRemove = nil
		}
		if v.deferRetry != nil {
			// cancel retrying this key
			_ = v.deferRetry.Stop()
			v.deferRetry = nil
		}
	}
	if !existed || start {
		if k.ctx != nil {
			v.start(k.ctx, v.exitedCh, false)
		}
	}
	return v.data, existed
}

// RemoveKey removes the given key from the set, if it exists.
// Returns if it existed.
func (k *Keyed[K, V]) RemoveKey(key K) bool {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	v, existed := k.routines[key]
	if existed {
		v.remove()
	}
	return existed
}

// SyncKeys synchronizes the list of running routines with the given list.
// If restart=true, restarts any routines in the failed state.
func (k *Keyed[K, V]) SyncKeys(keys []K, restart bool) (added, removed []K) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	if k.ctx != nil && k.ctx.Err() != nil {
		k.ctx = nil
	}

	routines := make(map[K]*runningRoutine[K, V], len(keys))
	for _, key := range keys {
		v := routines[key]
		if v != nil {
			// already processed
			continue
		}

		v, existed := k.routines[key]
		if !existed {
			routine, data := k.ctorCb(key)
			v = newRunningRoutine(k, key, routine, data, k.backoffFactory)
			k.routines[key] = v
			added = append(added, key)
		}

		routines[key] = v
		if (!existed || restart) && k.ctx != nil {
			v.start(k.ctx, v.exitedCh, false)
		}
	}
	for key, rr := range k.routines {
		if _, ok := routines[key]; !ok {
			removed = append(removed, key)
			rr.remove()
		}
	}

	return
}

// GetKey returns the value for the given key and existed.
func (k *Keyed[K, V]) GetKey(key K) (V, bool) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	v, existed := k.routines[key]
	if !existed {
		var empty V
		return empty, false
	}

	return v.data, true
}

// ResetRoutine resets the given routine after checking the condition functions.
// If any of the conds functions return true, resets the instance.
//
// Resetting the instance constructs a new Routine and data with the constructor.
// Note: this will overwrite the existing Data, if present!
// In most cases RestartRoutine is actually what you want.
//
// If len(conds) == 0, always resets the given key.
func (k *Keyed[K, V]) ResetRoutine(key K, conds ...func(K, V) bool) (existed bool, reset bool) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	return k.resetRoutineLocked(key, conds...)
}

// ResetAllRoutines resets all routines after checking the condition functions.
// If any of the conds functions return true for an instance, resets the instance.
//
// Resetting the instance constructs a new Routine and data with the constructor.
// Note: this will overwrite the existing Data, if present!
// In most cases RestartRoutine is actually what you want.
//
// If len(conds) == 0, always resets the keys.
func (k *Keyed[K, V]) ResetAllRoutines(conds ...func(K, V) bool) (resetCount, totalCount int) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	totalCount = len(k.routines)
	for key := range k.routines {
		if existed, reset := k.resetRoutineLocked(key, conds...); existed && reset {
			resetCount++
		}
	}

	return
}

// resetRoutineLocked resets the given routine while mtx is locked.
func (k *Keyed[K, V]) resetRoutineLocked(key K, conds ...func(K, V) bool) (existed bool, reset bool) {
	if k.ctx != nil && k.ctx.Err() != nil {
		k.ctx = nil
	}

	v, existed := k.routines[key]
	if !existed {
		return false, false
	}

	anyMatched := len(conds) == 0
	for _, cond := range conds {
		if cond != nil && cond(key, v.data) {
			anyMatched = true
			break
		}
	}
	if !anyMatched {
		return true, false
	}

	if v.ctxCancel != nil {
		v.ctxCancel()
	}
	prevExitedCh := v.exitedCh
	routine, data := k.ctorCb(key)
	v = newRunningRoutine(k, key, routine, data, k.backoffFactory)
	k.routines[key] = v
	if k.ctx != nil {
		v.start(k.ctx, prevExitedCh, false)
	}

	return true, true
}

// RestartRoutine restarts the given routine after checking the condition functions.
// If any return true, and the routine is running, restarts the instance.
//
// If len(conds) == 0, always resets the given key.
func (k *Keyed[K, V]) RestartRoutine(key K, conds ...func(K, V) bool) (existed bool, reset bool) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	return k.restartRoutineLocked(key, conds...)
}

// RestartAllRoutines restarts all routines after checking the condition functions.
// If any return true, and the routine is running, restarts the instance.
//
// If len(conds) == 0, always resets the keys.
func (k *Keyed[K, V]) RestartAllRoutines(conds ...func(K, V) bool) (restartedCount, totalCount int) {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	totalCount = len(k.routines)
	for key := range k.routines {
		if existed, reset := k.restartRoutineLocked(key, conds...); existed && reset {
			restartedCount++
		}
	}

	return
}

// resetRoutineLocked restarts the given routine while mtx is locked.
func (k *Keyed[K, V]) restartRoutineLocked(key K, conds ...func(K, V) bool) (existed bool, reset bool) {
	if k.ctx != nil && k.ctx.Err() != nil {
		k.ctx = nil
	}

	v, existed := k.routines[key]
	if !existed {
		return false, false
	}
	if k.ctx == nil {
		return true, false
	}

	anyMatched := len(conds) == 0
	for _, cond := range conds {
		if cond != nil && cond(key, v.data) {
			anyMatched = true
			break
		}
	}
	if !anyMatched {
		return true, false
	}

	if v.ctxCancel != nil {
		v.ctxCancel()
		v.ctxCancel = nil
	}
	if k.ctx != nil {
		prevExitedCh := v.exitedCh
		v.start(k.ctx, prevExitedCh, true)
	}

	return true, true
}
