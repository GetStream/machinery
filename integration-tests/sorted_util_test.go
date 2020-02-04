package integration_test

import (
	"reflect"
	"sort"
	"testing"
)

func requireSortedEqual(t *testing.T, expected, actual []int64) {
	sort.Slice(actual, func(i, j int) bool {
		return actual[i] < actual[j]
	})

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf(
			"expected results = %v, actual results = %v",
			expected,
			actual,
		)
	}
}
