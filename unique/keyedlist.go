package unique

import (
	"golang.org/x/exp/maps"
)

// KeyedList watches a list of values for changes.
//
// getKey gets the unique key for the value.
// cmp checks if two values are equal. if equal, the old version of the value is used.
//
// changed is called when a value is added, removed, or changed
//
// K is the key type
// V is the value type
type KeyedList[K, V comparable] struct {
	getKey  func(val V) K
	cmp     func(k K, a, b V) bool
	changed func(k K, v V, added, removed bool)
	vals    map[K]V
}

// NewKeyedList constructs a new KeyedList.
func NewKeyedList[K, V comparable](
	getKey func(v V) K,
	cmp func(k K, a, b V) bool,
	changed func(k K, v V, added, removed bool),
	initial []V,
) *KeyedList[K, V] {
	vals := make(map[K]V, len(initial))
	for _, v := range initial {
		k := getKey(v)
		vals[k] = v
	}

	return &KeyedList[K, V]{
		getKey:  getKey,
		cmp:     cmp,
		changed: changed,
		vals:    vals,
	}
}

// GetKeys returns the list of keys stored in the list.
func (l *KeyedList[K, V]) GetKeys() []K {
	return maps.Keys(l.vals)
}

// GetValues returns the list of values stored in the list.
func (l *KeyedList[K, V]) GetValues() []V {
	return maps.Values(l.vals)
}

// SetValues sets the list of values contained within the KeyedList.
//
// Values that do not appear in the list are removed.
// Values that are identical to their existing values are ignored.
// Values that change or are added are stored.
func (l *KeyedList[K, V]) SetValues(vals ...V) {
	prevKeys := l.GetKeys()
	notSeen := make(map[K]struct{}, len(prevKeys))
	for _, prevKey := range prevKeys {
		notSeen[prevKey] = struct{}{}
	}

	for _, v := range vals {
		k := l.getKey(v)
		delete(notSeen, k)
		existing, ok := l.vals[k]
		if ok {
			// changed
			if !l.cmp(k, v, existing) {
				l.vals[k] = v
				l.changed(k, v, false, false)
			}
		} else {
			// added
			l.vals[k] = v
			l.changed(k, v, true, false)
		}
	}

	// remove not seen vals
	for k := range notSeen {
		oldVal := l.vals[k]
		delete(l.vals, k)
		l.changed(k, oldVal, false, true)
	}
}

// AppendValues appends the given values to the list, deduplicating by key.
//
// Values that are identical to their existing values are ignored.
// Values that change or are added are stored.
func (l *KeyedList[K, V]) AppendValues(vals ...V) {
	for _, v := range vals {
		k := l.getKey(v)
		existing, ok := l.vals[k]
		if ok {
			// changed
			if !l.cmp(k, v, existing) {
				l.vals[k] = v
				l.changed(k, v, false, false)
			}
		} else {
			// added
			l.vals[k] = v
			l.changed(k, v, true, false)
		}
	}
}

// RemoveValues removes the given values from the list by key.
//
// Ignores values that were not in the list.
func (l *KeyedList[K, V]) RemoveValues(vals ...V) {
	for _, v := range vals {
		k := l.getKey(v)
		existing, ok := l.vals[k]
		if ok {
			// removed
			delete(l.vals, k)
			l.changed(k, existing, false, true)
		}
	}
}

// RemoveKeys removes the given keys from the list.
//
// Ignores values that were not in the list.
func (l *KeyedList[K, V]) RemoveKeys(keys ...K) {
	for _, k := range keys {
		v, ok := l.vals[k]
		if ok {
			// removed
			delete(l.vals, k)
			l.changed(k, v, false, true)
		}
	}
}
