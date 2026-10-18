package assert_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k-shimada-e/equify/require"
)

func TestRequireSuccessAndFailure(t *testing.T) {
	x, y := identity{1, "x"}, identity{1, "y"}
	require.Equal(t, x, y)
	require.Equalf(t, x, y, "id=%d", 1)
	require.New(t).Equal(x, y)
	require.New(t).Equalf(x, y, "id=%d", 1)
	require.NewWithOptions(t, cmpopts.EquateEmpty()).Equal([]int(nil), []int{})
	require.NoError(t, nil)
	require.New(t).Len([]int{1}, 1)
	cases := []struct {
		name string
		run  func(*recordingT)
	}{
		{"Equal", func(r *recordingT) { require.Equal(r, 1, 2) }},
		{"Equalf", func(r *recordingT) { require.Equalf(r, 1, 2, "case=%d", 42) }},
		{"NotEqual", func(r *recordingT) { require.NotEqual(r, x, y) }},
		{"NotEqualf", func(r *recordingT) { require.NotEqualf(r, x, y, "case=%d", 42) }},
		{"method Equal", func(r *recordingT) { require.New(r).Equal(1, 2) }},
		{"method Equalf", func(r *recordingT) { require.New(r).Equalf(1, 2, "case=%d", 42) }},
		{"method NotEqual", func(r *recordingT) { require.New(r).NotEqual(x, y) }},
		{"method NotEqualf", func(r *recordingT) { require.New(r).NotEqualf(x, y, "case=%d", 42) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &recordingT{}
			func() {
				defer func() {
					if p := recover(); p != (stop{}) {
						t.Fatalf("unexpected panic: %v", p)
					}
				}()
				tc.run(r)
				r.continued = true
			}()
			if !r.stopped || r.continued || len(r.messages) == 0 {
				t.Fatalf("require did not fail immediately: %+v", r)
			}
		})
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
