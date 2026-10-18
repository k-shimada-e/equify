package assert_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k-shimada-e/equify/assert"
	"github.com/k-shimada-e/equify/require"
)

func TestComparisonOptions(t *testing.T) {
	a := assert.NewWithOptions(t, cmpopts.EquateEmpty())
	a.Equal([]int(nil), []int{})
	if assert.NewWithOptions(&recordingT{}, cmpopts.EquateEmpty()).NotEqual([]int{}, []int(nil)) {
		t.Fatal("options not applied to NotEqual")
	}
}

func TestOptionsAreCopiedAndScoped(t *testing.T) {
	opts := []cmp.Option{cmpopts.EquateEmpty()}
	r := &recordingT{}
	a := assert.NewWithOptions(r, opts...)
	q := require.NewWithOptions(r, opts...)
	// Replacing an element in the caller's slice must not reconfigure instances.
	opts[0] = cmpopts.SortSlices(func(x, y int) bool { return x < y })
	if !a.Equalf([]int(nil), []int{}, "equal %d", 1) {
		t.Fatal("Equalf lost the copied option")
	}
	if a.NotEqualf([]int(nil), []int{}, "not equal %d", 2) {
		t.Fatal("NotEqualf lost the copied option")
	}
	q.Equalf([]int(nil), []int{}, "require %d", 3)
	if assert.Equal(&recordingT{}, []int(nil), []int{}) {
		t.Fatal("instance options leaked into package functions")
	}
	if a.EqualValues([]int(nil), []int{}) {
		t.Fatal("instance options changed a delegated assertion")
	}
	if assert.EqualWithOptions(&recordingT{}, []int{2, 1}, []int{1, 2}, nil) {
		t.Fatal("nil options changed slice ordering")
	}
	if !assert.EqualWithOptions(t, []int{2, 1}, []int{1, 2}, opts) {
		t.Fatal("explicit sort option was ignored")
	}
	if assert.NotEqualWithOptions(&recordingT{}, []int{2, 1}, []int{1, 2}, opts) {
		t.Fatal("NotEqualWithOptions ignored the sort option")
	}
}
