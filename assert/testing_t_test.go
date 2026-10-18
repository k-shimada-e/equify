package assert_test

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/k-shimada-e/equify/assert"
	"github.com/k-shimada-e/equify/require"
)

// Use a failing child test to verify testing.T's actual failure location.
func TestFailureLocation(t *testing.T) {
	if os.Getenv("EQUIFY_FAILURE_CHILD") == "1" {
		_, _, line, _ := runtime.Caller(0)
		t.Logf("expected location: testing_t_test.go:%d:", line+2)
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
