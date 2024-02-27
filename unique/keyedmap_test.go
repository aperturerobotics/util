package unique

import (
	"fmt"
	"reflect"
	"testing"
)

type testValue struct {
	ID   int
	Name string
}

func TestKeyedMap(t *testing.T) {
	var changes []struct {
		key     int
		value   testValue
		added   bool
		removed bool
	}

	// Define a comparison function
	cmp := func(k int, a, b testValue) bool {
		return a.ID == b.ID && a.Name == b.Name
	}

	// Define a change tracking function
	changed := func(k int, v testValue, added, removed bool) {
		changes = append(changes, struct {
			key     int
			value   testValue
			added   bool
			removed bool
		}{k, v, added, removed})
	}

	initial := map[int]testValue{
		1: {1, "Alice"},
		2: {2, "Bob"},
	}

	kMap := NewKeyedMap(cmp, changed, initial)

	// Test adding, updating, and removing values
	t.Run("SetValues - Add, Update, Remove", func(t *testing.T) {
		changes = nil // Reset changes tracking

		newValues := map[int]testValue{
			2: {2, "Bobby"},
			3: {3, "Charlie"},
		}
		kMap.SetValues(newValues)

		expectedChangesMap := map[string]struct {
			value   testValue
			added   bool
			removed bool
		}{
			"2-false-false": {newValues[2], false, false}, // updated
			"1-false-true":  {initial[1], false, true},    // removed
			"3-true-false":  {newValues[3], true, false},  // added
		}

		for _, change := range changes {
			key := formatChangeKey(change.key, change.added, change.removed)
			expected, ok := expectedChangesMap[key]
			if !ok || !reflect.DeepEqual(change.value, expected.value) {
				t.Errorf("Unexpected or incorrect change for key %d: got %+v, want %+v", change.key, change, expected)
			}
		}
	})
}

func formatChangeKey(key int, added, removed bool) string {
	return fmt.Sprintf("%d-%t-%t", key, added, removed)
}
