package unique

import (
	"maps"
	"slices"
)

// KeyedMap watches a map of values for changes.
//
// cmp checks if two values are equal. if equal, the old version of the value is used.
//
// changed is called when a value is added, removed, or changed
//
// K is the key type
// V is the value type
type KeyedMap[K, V comparable] struct {
	cmp     func(k K, a, b V) bool
	changed func(k K, v V, added, removed bool)
	vals    map[K]V
}

// NewKeyedMap constructs a new KeyedMap.
func NewKeyedMap[K, V comparable](
	cmp func(k K, a, b V) bool,
	changed func(k K, v V, added, removed bool),
	initial map[K]V,
) *KeyedMap[K, V] {
	vals := make(map[K]V, len(initial))
	maps.Copy(vals, initial)

	return &KeyedMap[K, V]{
		cmp:     cmp,
		changed: changed,
		vals:    vals,
	}
}

// GetKeys returns the list of keys stored in the map.
func (l *KeyedMap[K, V]) GetKeys() []K {
	return slices.Collect(maps.Keys(l.vals))
}

// GetValues returns the list of values stored in the map.
func (l *KeyedMap[K, V]) GetValues() []V {
	return slices.Collect(maps.Values(l.vals))
}

// SetValues sets the list of values contained within the KeyedMap.
//
// Values that do not appear in the list are removed.
// Values that are identical to their existing values are ignored.
// Values that change or are added are stored.
func (l *KeyedMap[K, V]) SetValues(vals map[K]V) {
	prevKeys := l.GetKeys()
	notSeen := make(map[K]struct{}, len(prevKeys))
	for _, prevKey := range prevKeys {
		notSeen[prevKey] = struct{}{}
	}

	for k, v := range vals {
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
func (l *KeyedMap[K, V]) AppendValues(vals map[K]V) {
	for k, v := range vals {
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

// RemoveKeys removes the given keys from the list.
//
// Ignores values that were not in the list.
func (l *KeyedMap[K, V]) RemoveKeys(keys ...K) {
	for _, k := range keys {
		v, ok := l.vals[k]
		if ok {
			// removed
			delete(l.vals, k)
			l.changed(k, v, false, true)
		}
	}
}
