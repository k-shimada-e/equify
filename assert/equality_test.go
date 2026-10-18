package assert_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/k-shimada-e/equify/assert"
	"github.com/k-shimada-e/equify/require"
)

type identity struct {
	ID    int
	cache string
}

func (x identity) Equal(y identity) bool { return x.ID == y.ID }

type private struct{ n int }

func TestEqualitySemantics(t *testing.T) {
	cases := []struct {
		name      string
		want, got interface{}
		equal     bool
	}{
		{"method", identity{1, "old"}, identity{1, "new"}, true},
		{"nested method", []identity{{1, "old"}}, []identity{{1, "new"}}, true},
		{"method mismatch", identity{1, "same"}, identity{2, "same"}, false},
		{"same instant", time.Unix(100, 0).UTC(), time.Unix(100, 0).In(time.FixedZone("offset", 3600)), true},
		{"private fields equal", private{1}, private{1}, true},
		{"private fields mismatch", private{1}, private{2}, false},
		{"type mismatch", int(1), int64(1), false},
		{"nil slice", []int(nil), []int{}, false},
		{"bytes", []byte{1, 2}, []byte{1, 2}, true},
		{"nil", nil, nil, true},
		{"typed nil", (*int)(nil), nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &recordingT{}
			if got := assert.Equal(r, tc.want, tc.got); got != tc.equal {
				t.Fatalf("Equal=%v, want %v", got, tc.equal)
			}
			if r.stopped {
				t.Fatal("assert stopped execution")
			}
			if (len(r.messages) == 0) != tc.equal {
				t.Fatalf("messages=%v", r.messages)
			}
			if got := assert.ObjectsAreEqual(tc.want, tc.got); got != tc.equal {
				t.Fatalf("ObjectsAreEqual=%v", got)
			}
			if got := assert.NotEqual(&recordingT{}, tc.want, tc.got); got == tc.equal {
				t.Fatalf("NotEqual=%v", got)
			}
		})
	}
}

func TestFormattedAndMethodAssertions(t *testing.T) {
	x, y := identity{1, "x"}, identity{1, "y"}
	r := &recordingT{}
	a := assert.New(r)
	if !a.Equal(x, y) || !a.Equalf(x, y, "id=%d", 1) {
		t.Fatal("methods ignored Equal")
	}
	if !assert.Equalf(r, x, y, "id=%d", 1) {
		t.Fatal("Equalf ignored Equal")
	}
	if a.NotEqual(x, y, "custom message") {
		t.Fatal("NotEqual ignored Equal")
	}
	if a.NotEqualf(x, y, "id=%d", 42) {
		t.Fatal("NotEqualf ignored Equal")
	}
	if assert.NotEqualf(r, x, y, "id=%d", 43) {
		t.Fatal("package NotEqualf ignored Equal")
	}
	if !strings.Contains(strings.Join(r.messages, "\n"), "id=42") {
		t.Fatal("formatted message lost")
	}
}

func TestFunctionArguments(t *testing.T) {
	var fn func()
	for _, equal := range []func(assert.TestingT, interface{}, interface{}, ...interface{}) bool{assert.Equal, assert.NotEqual} {
		r := &recordingT{}
		if equal(r, fn, fn) || len(r.messages) != 1 {
			t.Fatal("function arguments must fail assertions")
		}
	}
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
