package assert_test

import (
	"fmt"
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
