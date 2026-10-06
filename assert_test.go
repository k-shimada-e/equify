package assert_test

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	assert "github.com/k-shimada-e/equify"
	"github.com/k-shimada-e/equify/require"
	"github.com/google/go-cmp/cmp/cmpopts"
	testify "github.com/stretchr/testify/assert"
)

type recordingT struct {
	messages  []string
	stopped   bool
	continued bool
}

func (r *recordingT) Errorf(format string, args ...interface{}) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}
func (r *recordingT) Helper() {}

type stop struct{}

func (r *recordingT) FailNow() { r.stopped = true; panic(stop{}) }

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

func TestDiffAndOptions(t *testing.T) {
	r := &recordingT{}
	assert.Equal(r, []int{1}, []int{2}, "case=%s", "diff")
	msg := strings.Join(r.messages, "\n")
	for _, part := range []string{"Not equal:", "expected: []int{1}", "actual  : []int{2}", "Diff:", "--- Expected", "+++ Actual", "case=diff"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %s", part, msg)
		}
	}
	a := assert.NewWithOptions(t, cmpopts.EquateEmpty())
	a.Equal([]int(nil), []int{})
	if assert.NewWithOptions(&recordingT{}, cmpopts.EquateEmpty()).NotEqual([]int{}, []int(nil)) {
		t.Fatal("options not applied to NotEqual")
	}
}

func errorBody(r *recordingT) string {
	_, body, _ := strings.Cut(strings.Join(r.messages, "\n"), "\tError:      \t")
	body = strings.ReplaceAll(body, "\t            \t", "")
	return strings.TrimSpace(body)
}

func TestTestifyStyleFailureSummaries(t *testing.T) {
	var fn func()
	cases := []struct {
		name      string
		want, got interface{}
	}{
		{"struct with Equal", identity{1, "old"}, identity{2, "new"}},
		{"different numeric types", int(1), int64(1)},
		{"duration", time.Second, 2 * time.Second},
		{"nil", nil, 1},
		{"invalid function", fn, fn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			custom, original := &recordingT{}, &recordingT{}
			assert.Equal(custom, tc.want, tc.got)
			testify.Equal(original, tc.want, tc.got)
			got, _, _ := strings.Cut(errorBody(custom), "\n\nDiff:")
			want, _, _ := strings.Cut(errorBody(original), "\n\nDiff:")
			if strings.TrimSpace(got) != strings.TrimSpace(want) {
				t.Fatalf("summary differs from testify:\ngot: %s\nwant: %s", got, want)
			}
		})
	}
	custom, original := &recordingT{}, &recordingT{}
	assert.NotEqual(custom, 1, 1)
	testify.NotEqual(original, 1, 1)
	if errorBody(custom) != errorBody(original) {
		t.Fatal("NotEqual failure text differs from testify")
	}
}

func TestEqualMethodDiffLayout(t *testing.T) {
	r := &recordingT{}
	assert.Equal(r, identity{1, "old"}, identity{2, "new"}, "user comparison")
	text := errorBody(r)
	for _, part := range []string{
		"expected: assert_test.identity{ID:1, cache:\"old\"}",
		"actual  : assert_test.identity{ID:2, cache:\"new\"}",
		"Diff:\n--- Expected\n+++ Actual\n",
		"ID: 1", "ID: 2", "user comparison",
	} {
		if !strings.Contains(text, part) {
			t.Fatalf("missing %q:\n%s", part, text)
		}
	}
}

func TestUnchangedAssertions(t *testing.T) {
	if !assert.EqualValues(t, int(1), int64(1)) {
		t.Fatal("numeric conversion changed")
	}
	assert.New(t).EqualValues(int(1), int64(1))
	assert.Contains(t, []string{"a"}, "a")
	assert.New(t).Len([]int{1}, 1)
	assert.NoError(t, nil)
	// Public callback aliases retain testify's function signatures.
	var f testify.ComparisonAssertionFunc = assert.Equal
	f(t, identity{1, "a"}, identity{1, "b"})
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

// Use a failing child test to verify testing.T's actual failure location.
func TestFailureLocation(t *testing.T) {
	if os.Getenv("EQUIFY_FAILURE_CHILD") == "1" {
		_, _, line, _ := runtime.Caller(0)
		t.Logf("expected location: assert_test.go:%d:", line+2)
		assert.Equal(t, 1, 2)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFailureLocation$", "-test.v")
	cmd.Env = append(os.Environ(), "EQUIFY_FAILURE_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("child test unexpectedly passed")
	}
	text := string(out)
	for _, line := range strings.Split(text, "\n") {
		if _, location, ok := strings.Cut(line, "expected location: "); ok {
			if !strings.Contains(text, "    "+strings.TrimSpace(location)+" \n") {
				t.Fatalf("wrong failure location:\n%s", text)
			}
			return
		}
	}
	t.Fatalf("child failed without location marker:\n%s", text)
}

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
