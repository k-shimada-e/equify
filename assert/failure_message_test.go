package assert_test

import (
	"strings"
	"testing"
	"time"

	"github.com/k-shimada-e/equify/assert"
	testify "github.com/stretchr/testify/assert"
)

func TestDiffOutput(t *testing.T) {
	r := &recordingT{}
	assert.Equal(r, []int{1}, []int{2}, "case=%s", "diff")
	msg := strings.Join(r.messages, "\n")
	for _, part := range []string{"Not equal:", "expected: []int{1}", "actual  : []int{2}", "Diff:", "--- Expected", "+++ Actual", "case=diff"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %s", part, msg)
		}
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

func TestLongFailureSummaryIsTruncated(t *testing.T) {
	r := &recordingT{}
	assert.Equal(r, strings.Repeat("x", 70_000), "short")
	text, _, _ := strings.Cut(errorBody(r), "\n\nDiff:")
	if !strings.Contains(text, "<... truncated>") || len(text) >= 70_000 {
		t.Fatal("long failure summary was not truncated")
	}
}
