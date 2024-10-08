package keyed

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// KeyedRefCount manages a list of running routines with reference counts.
type KeyedRefCount[K comparable, V any] struct {
	// keyed is the underlying keyed controller
	keyed *Keyed[K, V]

	// mtx guards below fields
	mtx sync.Mutex
	// refs is the list of keyed refs.
	refs map[K][]*KeyedRef[K, V]
}

// KeyedRef is a reference to a key.
type KeyedRef[K comparable, V any] struct {
	rc  *KeyedRefCount[K, V]
	key K
	rel atomic.Bool
}

// Release releases the reference.
func (k *KeyedRef[K, V]) Release() {
	if k == nil {
		return
	}
	if k.rel.Swap(true) {
		return
	}
	k.rc.mtx.Lock()
	refs := k.rc.refs[k.key]
	for i := 0; i < len(refs); i++ {
		if refs[i] == k {
			refs[i] = refs[len(refs)-1]
			refs[len(refs)-1] = nil
			refs = refs[:len(refs)-1]

			if len(refs) == 0 {
				delete(k.rc.refs, k.key)
				_ = k.rc.keyed.RemoveKey(k.key)
			} else {
				k.rc.refs[k.key] = refs
			}
			break
		}
	}
	k.rc.mtx.Unlock()
}

// NewKeyedRefCount constructs a new Keyed execution manager with reference counting.
// Note: routines won't start until SetContext is called.
func NewKeyedRefCount[K comparable, V any](
	ctorCb func(key K) (Routine, V),
	opts ...Option[K, V],
) *KeyedRefCount[K, V] {
	return &KeyedRefCount[K, V]{
		keyed: NewKeyed(ctorCb, opts...),
		refs:  make(map[K][]*KeyedRef[K, V]),
	}
}

// NewKeyedRefCountWithLogger constructs a new Keyed execution manager with reference counting.
// Logs when a controller exits without being removed from the Keys set.
// Note: routines won't start until SetContext is called.
func NewKeyedRefCountWithLogger[K comparable, V any](
	ctorCb func(key K) (Routine, V),
	le *logrus.Entry,
	opts ...Option[K, V],
) *KeyedRefCount[K, V] {
	return &KeyedRefCount[K, V]{
		keyed: NewKeyedWithLogger(ctorCb, le, opts...),
		refs:  make(map[K][]*KeyedRef[K, V]),
	}
}

// SetContext updates the root context, restarting all running routines.
// if restart is true, all errored routines also restart
//
// nil context is valid and will shutdown the routines.
func (k *KeyedRefCount[K, V]) SetContext(ctx context.Context, restart bool) {
	k.keyed.SetContext(ctx, restart)
}

// ClearContext clears the context and shuts down all routines.
func (k *KeyedRefCount[K, V]) ClearContext() {
	k.keyed.ClearContext()
}

// GetKeys returns the list of keys registered with the Keyed instance.
func (k *KeyedRefCount[K, V]) GetKeys() []K {
	return k.keyed.GetKeys()
}

// GetKeysWithData returns the keys and the data for the keys.
func (k *KeyedRefCount[K, V]) GetKeysWithData() []KeyWithData[K, V] {
	return k.keyed.GetKeysWithData()
}

// GetKey returns the value for the given key and if it existed.
func (k *KeyedRefCount[K, V]) GetKey(key K) (V, bool) {
	return k.keyed.GetKey(key)
}

// ResetRoutine resets the given routine after checking the condition functions.
// If any return true, resets the instance.
//
// If len(conds) == 0, always resets the given key.
func (k *KeyedRefCount[K, V]) ResetRoutine(key K, conds ...func(K, V) bool) (existed bool, reset bool) {
	return k.keyed.ResetRoutine(key, conds...)
}

// RestartRoutine restarts the given routine after checking the condition functions.
// If any return true, and the routine is running, restarts the instance.
//
// If len(conds) == 0, always resets the given key.
func (k *KeyedRefCount[K, V]) RestartRoutine(key K, conds ...func(K, V) bool) (existed bool, reset bool) {
	return k.keyed.RestartRoutine(key, conds...)
}

// ResetAllRoutines resets all routines after checking the condition functions.
// If any of the conds functions return true for an instance, resets the instance.
//
// Resetting the instance constructs a new Routine and data with the constructor.
// Note: this will overwrite the existing Data, if present!
// In most cases RestartRoutine is actually what you want.
//
// If len(conds) == 0, always resets the keys.
func (k *KeyedRefCount[K, V]) ResetAllRoutines(conds ...func(K, V) bool) (resetCount, totalCount int) {
	return k.keyed.ResetAllRoutines(conds...)
}

// RestartAllRoutines restarts all routines after checking the condition functions.
// If any return true, and the routine is running, restarts the instance.
//
// If len(conds) == 0, always resets the keys.
func (k *KeyedRefCount[K, V]) RestartAllRoutines(conds ...func(K, V) bool) (restartedCount, totalCount int) {
	return k.keyed.RestartAllRoutines(conds...)
}

// RemoveKey removes all references to a key deleting it from the set and returning if it existed.
//
// Returns if the key existed.
func (k *KeyedRefCount[K, V]) RemoveKey(key K) bool {
	k.mtx.Lock()
	defer k.mtx.Unlock()

	// clear all refs to the key
	for _, ref := range k.refs[key] {
		// mark as released
		ref.rel.Store(true)
	}
	delete(k.refs, key)

	// return if the key existed
	return k.keyed.RemoveKey(key)
}

// AddKeyRef adds a reference to the given key.
// Returns if the key already existed or not.
func (k *KeyedRefCount[K, V]) AddKeyRef(key K) (ref *KeyedRef[K, V], data V, existed bool) {
	k.mtx.Lock()
	refs := k.refs[key]
	nref := &KeyedRef[K, V]{rc: k, key: key}
	data, existed = k.keyed.SetKey(key, true)
	refs = append(refs, nref)
	k.refs[key] = refs
	k.mtx.Unlock()
	return nref, data, existed
}
