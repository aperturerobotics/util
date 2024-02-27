package unique

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type testKeyedListValue struct {
	id   int
	name string
}

func TestKeyedList(t *testing.T) {
	// Setup getKey, cmp, and changed functions
	getKey := func(v testKeyedListValue) int {
		return v.id
	}

	cmp := func(k int, a, b testKeyedListValue) bool {
		return a.name == b.name
	}

	var changes []struct {
		key     int
		value   testKeyedListValue
		added   bool
		removed bool
	}
	changed := func(k int, v testKeyedListValue, added, removed bool) {
		changes = append(changes, struct {
			key     int
			value   testKeyedListValue
			added   bool
			removed bool
		}{k, v, added, removed})
	}

	initial := []testKeyedListValue{{1, "Alice"}, {2, "Bob"}}

	// Create new KeyedList
	list := NewKeyedList[int, testKeyedListValue](getKey, cmp, changed, initial)

	t.Run("GetKeys", func(t *testing.T) {
		expectedKeys := []int{1, 2}
		keys := list.GetKeys()
		sort.Ints(keys) // Ensure the order for comparison
		if !reflect.DeepEqual(keys, expectedKeys) {
			t.Errorf("expected keys %v, got %v", expectedKeys, keys)
		}
	})

	t.Run("GetValues", func(t *testing.T) {
		expectedValues := []testKeyedListValue{{1, "Alice"}, {2, "Bob"}}
		values := list.GetValues()
		sort.Slice(values, func(i, j int) bool {
			return values[i].id < values[j].id
		})
		if !reflect.DeepEqual(values, expectedValues) {
			t.Errorf("expected values %v, got %v", expectedValues, values)
		}
	})

	t.Run("SetValues - Add, Update, Remove", func(t *testing.T) {
		// Reset changes tracking
		changes = nil

		newValues := []testKeyedListValue{{2, "Bobby"}, {3, "Charlie"}}
		list.SetValues(newValues...)

		// Prepare a map to track the occurrence of changes
		changesMap := make(map[string]struct {
			key     int
			value   testKeyedListValue
			added   bool
			removed bool
		})
		for _, change := range changes {
			key := fmt.Sprintf("%d-%t-%t", change.key, change.added, change.removed)
			changesMap[key] = change
		}

		// Define a function to check for an expected change
		checkForChange := func(expected struct {
			key     int
			value   testKeyedListValue
			added   bool
			removed bool
		}) {
			key := fmt.Sprintf("%d-%t-%t", expected.key, expected.added, expected.removed)
			if change, exists := changesMap[key]; !exists || !reflect.DeepEqual(change.value, expected.value) {
				t.Errorf("change for key %s not as expected: %+v", key, change)
			}
		}

		// Check for each expected change
		expectedChanges := []struct {
			key     int
			value   testKeyedListValue
			added   bool
			removed bool
		}{
			{2, newValues[0], false, false}, // updated
			{1, initial[0], false, true},    // removed
			{3, newValues[1], true, false},  // added
		}
		for _, expectedChange := range expectedChanges {
			checkForChange(expectedChange)
		}
	})

	t.Run("AppendValues - Add and Update", func(t *testing.T) {
		// Reset changes tracking
		changes = nil

		appendValues := []testKeyedListValue{{3, "Charlie"}, {4, "Dana"}}
		list.AppendValues(appendValues...)

		// Check for expected changes
		expectedChanges := []struct {
			key     int
			value   testKeyedListValue
			added   bool
			removed bool
		}{
			// Since Charlie is already in the list with the same value, no change should be triggered for it
			{4, appendValues[1], true, false}, // added
		}
		if !reflect.DeepEqual(changes, expectedChanges) {
			t.Errorf("expected changes %v, got %v", expectedChanges, changes)
		}
	})

	t.Run("RemoveValues and RemoveKeys", func(t *testing.T) {
		// Reset changes tracking
		changes = nil

		// Remove by value and key
		list.RemoveValues(testKeyedListValue{2, "Bobby"})
		list.RemoveKeys(4) // Dana

		expectedChanges := []struct {
			key     int
			value   testKeyedListValue
			added   bool
			removed bool
		}{
			{2, testKeyedListValue{2, "Bobby"}, false, true}, // removed by value
			{4, testKeyedListValue{4, "Dana"}, false, true},  // removed by key
		}
		if !reflect.DeepEqual(changes, expectedChanges) {
			t.Errorf("expected changes %v, got %v", expectedChanges, changes)
		}
	})
}
