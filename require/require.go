// Package require provides testify-compatible assertions that stop on failure.
package require

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/k-shimada-e/equify/assert"
	testify "github.com/stretchr/testify/require"
)

type Assertions struct {
	*testify.Assertions
	t        TestingT
	equality *assert.Assertions
}

func New(t TestingT) *Assertions { return NewWithOptions(t) }

func NewWithOptions(t TestingT, opts ...cmp.Option) *Assertions {
	return &Assertions{Assertions: testify.New(t), t: t, equality: assert.NewWithOptions(t, opts...)}
}

type helperT interface{ Helper() }
type noHelper struct{}

func (noHelper) Helper() {}

func markHelper(t TestingT) helperT {
	if h, ok := t.(interface{ Helper() }); ok {
		return h
	}
	return noHelper{}
}

func Equal(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) {
	markHelper(t).Helper()
	if !assert.Equal(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func Equalf(t TestingT, expected, actual interface{}, msg string, args ...interface{}) {
	markHelper(t).Helper()
	Equal(t, expected, actual, fmt.Sprintf(msg, args...))
}

func NotEqual(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) {
	markHelper(t).Helper()
	if !assert.NotEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func NotEqualf(t TestingT, expected, actual interface{}, msg string, args ...interface{}) {
	markHelper(t).Helper()
	NotEqual(t, expected, actual, fmt.Sprintf(msg, args...))
}

func (a *Assertions) Equal(expected, actual interface{}, msgAndArgs ...interface{}) {
	markHelper(a.t).Helper()
	if !a.equality.Equal(expected, actual, msgAndArgs...) {
		a.t.FailNow()
	}
}

func (a *Assertions) Equalf(expected, actual interface{}, msg string, args ...interface{}) {
	markHelper(a.t).Helper()
	a.Equal(expected, actual, fmt.Sprintf(msg, args...))
}

func (a *Assertions) NotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	markHelper(a.t).Helper()
	if !a.equality.NotEqual(expected, actual, msgAndArgs...) {
		a.t.FailNow()
	}
}

func (a *Assertions) NotEqualf(expected, actual interface{}, msg string, args ...interface{}) {
	markHelper(a.t).Helper()
	a.NotEqual(expected, actual, fmt.Sprintf(msg, args...))
}
