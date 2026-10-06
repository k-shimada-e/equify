package assert_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	assert "github.com/k-shimada-e/equify"
	"github.com/k-shimada-e/equify/require"
)

// TestingT does not require Helper; minimal implementations must work too.
type minimalT struct {
	messages int
	stopped  bool
}

func (r *minimalT) Errorf(string, ...interface{}) { r.messages++ }
func (r *minimalT) FailNow()                      { r.stopped = true }

func TestWithoutHelper(t *testing.T) {
	r := &minimalT{}
	if !assert.Equal(r, 1, 1) || assert.NotEqual(r, 1, 1) {
		t.Fatal("unexpected assertion results without Helper")
	}
	require.Equal(r, 1, 2)
	if r.messages != 2 || !r.stopped {
		t.Fatalf("failure was not reported: %+v", r)
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

func TestRequireNotEqualSuccess(t *testing.T) {
	require.NotEqual(t, 1, 2)
	require.NotEqualf(t, 1, 2, "case %d", 1)
	require.New(t).NotEqual(1, 2)
	require.New(t).NotEqualf(1, 2, "case %d", 2)
	require.NewWithOptions(t, cmpopts.EquateEmpty()).NotEqual([]int{}, []int{1})
}

func TestRequireOptionsFailure(t *testing.T) {
	r := &recordingT{}
	defer func() {
		if p := recover(); p != (stop{}) {
			t.Fatalf("expected FailNow, got %v", p)
		}
		if !r.stopped || !strings.Contains(strings.Join(r.messages, "\n"), "case=42") {
			t.Fatalf("failure or formatted message lost: %+v", r)
		}
	}()
	require.NewWithOptions(r, cmpopts.EquateEmpty()).NotEqualf([]int(nil), []int{}, "case=%d", 42)
	t.Fatal("require continued after failure")
}

func TestCyclicValues(t *testing.T) {
	type node struct {
		Value int
		Next  *node
	}
	x, y := &node{Value: 1}, &node{Value: 1}
	x.Next, y.Next = x, y
	if !assert.Equal(t, x, y) || !assert.ObjectsAreEqual(x, y) {
		t.Fatal("equal cycles were not recognized")
	}
	y.Value = 2
	if !assert.NotEqual(t, x, y) || assert.Equal(&recordingT{}, x, y) {
		t.Fatal("different cycles were not recognized")
	}
}

func TestComparisonPanicsPropagate(t *testing.T) {
	const sentinel = "comparison failed"
	compare := cmp.Comparer(func(x, y int) bool { panic(sentinel) })
	for name, run := range map[string]func(){
		"Equal":    func() { assert.EqualWithOptions(&recordingT{}, 1, 2, []cmp.Option{compare}) },
		"NotEqual": func() { assert.NotEqualWithOptions(&recordingT{}, 1, 2, []cmp.Option{compare}) },
		"require":  func() { require.NewWithOptions(&recordingT{}, compare).Equal(1, 2) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != sentinel {
					t.Fatalf("comparison panic changed: %v", p)
				}
			}()
			run()
			t.Fatal("comparison panic was swallowed")
		})
	}
}

func TestLongFailureSummaryIsTruncated(t *testing.T) {
	r := &recordingT{}
	assert.Equal(r, strings.Repeat("x", 70_000), "short")
	text, _, _ := strings.Cut(errorBody(r), "\n\nDiff:")
	if !strings.Contains(text, "<... truncated>") || len(text) >= 70_000 {
		t.Fatal("long failure summary was not truncated")
	}
}
