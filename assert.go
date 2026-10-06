// Package assert provides testify-compatible assertions with go-cmp equality.
package assert

import (
	"bufio"
	"fmt"
	"reflect"
	"time"

	"github.com/google/go-cmp/cmp"
	testify "github.com/stretchr/testify/assert"
)

//go:generate go run ./internal/gen

// Assertions preserves testify's methods and overrides Equal and NotEqual.
type Assertions struct {
	*testify.Assertions
	t    TestingT
	opts []cmp.Option
}

func New(t TestingT) *Assertions { return NewWithOptions(t) }

// NewWithOptions applies comparison options to this instance's equality methods.
// Other methods retain testify's behavior.
func NewWithOptions(t TestingT, opts ...cmp.Option) *Assertions {
	return &Assertions{Assertions: testify.New(t), t: t, opts: append([]cmp.Option(nil), opts...)}
}

type helperT interface{ Helper() }
type noHelper struct{}

func (noHelper) Helper() {}

// The caller invokes Helper directly so testing marks the assertion's frame.
func markHelper(t TestingT) helperT {
	if h, ok := t.(interface{ Helper() }); ok {
		return h
	}
	return noHelper{}
}

// comparisonOptions preserves testify's comparison of unexported fields.
// Equal methods still take precedence, following go-cmp's comparison rules.
func comparisonOptions(opts []cmp.Option) []cmp.Option {
	return append([]cmp.Option{cmp.Exporter(func(_ reflect.Type) bool { return true })}, opts...)
}

// Equal compares values using go-cmp, including supported Equal methods.
func Equal(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	markHelper(t).Helper()
	return EqualWithOptions(t, expected, actual, nil, msgAndArgs...)
}

// EqualWithOptions accepts explicit go-cmp options without changing Equal's signature.
func EqualWithOptions(t TestingT, expected, actual interface{}, opts []cmp.Option, msgAndArgs ...interface{}) bool {
	markHelper(t).Helper()
	if isFunction(expected) || isFunction(actual) {
		return testify.Fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (cannot take func type as argument)", expected, actual), msgAndArgs...)
	}
	if diff := cmp.Diff(expected, actual, comparisonOptions(opts)...); diff != "" {
		e, a := formatUnequalValues(expected, actual)
		return testify.Fail(t, fmt.Sprintf("Not equal: \nexpected: %s\nactual  : %s\n\nDiff:\n--- Expected\n+++ Actual\n%s", e, a, diff), msgAndArgs...)
	}
	return true
}

func Equalf(t TestingT, expected, actual interface{}, msg string, args ...interface{}) bool {
	markHelper(t).Helper()
	return Equal(t, expected, actual, fmt.Sprintf(msg, args...))
}

func NotEqual(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	markHelper(t).Helper()
	return NotEqualWithOptions(t, expected, actual, nil, msgAndArgs...)
}

func NotEqualWithOptions(t TestingT, expected, actual interface{}, opts []cmp.Option, msgAndArgs ...interface{}) bool {
	markHelper(t).Helper()
	if isFunction(expected) || isFunction(actual) {
		return testify.Fail(t, fmt.Sprintf("Invalid operation: %#v != %#v (cannot take func type as argument)", expected, actual), msgAndArgs...)
	}
	if cmp.Equal(expected, actual, comparisonOptions(opts)...) {
		return testify.Fail(t, fmt.Sprintf("Should not be: %#v\n", actual), msgAndArgs...)
	}
	return true
}

func NotEqualf(t TestingT, expected, actual interface{}, msg string, args ...interface{}) bool {
	markHelper(t).Helper()
	return NotEqual(t, expected, actual, fmt.Sprintf(msg, args...))
}

func ObjectsAreEqual(expected, actual interface{}) bool {
	return cmp.Equal(expected, actual, comparisonOptions(nil)...)
}

func isFunction(value interface{}) bool {
	return value != nil && reflect.TypeOf(value).Kind() == reflect.Func
}

// Adapted from testify's formatUnequalValues (MIT; see LICENSE).
// Match testify's value summaries while leaving equality and diffs to go-cmp.
func formatUnequalValues(expected, actual interface{}) (string, string) {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		return fmt.Sprintf("%T(%s)", expected, formatValue(expected)),
			fmt.Sprintf("%T(%s)", actual, formatValue(actual))
	}
	if _, ok := expected.(time.Duration); ok {
		return fmt.Sprintf("%v", expected), fmt.Sprintf("%v", actual)
	}
	return formatValue(expected), formatValue(actual)
}

func formatValue(value interface{}) string {
	s := fmt.Sprintf("%#v", value)
	const maxLength = bufio.MaxScanTokenSize - 100
	if len(s) > maxLength {
		return s[:maxLength] + "<... truncated>"
	}
	return s
}

func (a *Assertions) Equal(expected, actual interface{}, msgAndArgs ...interface{}) bool {
	markHelper(a.t).Helper()
	return EqualWithOptions(a.t, expected, actual, a.opts, msgAndArgs...)
}

func (a *Assertions) Equalf(expected, actual interface{}, msg string, args ...interface{}) bool {
	markHelper(a.t).Helper()
	return a.Equal(expected, actual, fmt.Sprintf(msg, args...))
}

func (a *Assertions) NotEqual(expected, actual interface{}, msgAndArgs ...interface{}) bool {
	markHelper(a.t).Helper()
	return NotEqualWithOptions(a.t, expected, actual, a.opts, msgAndArgs...)
}

func (a *Assertions) NotEqualf(expected, actual interface{}, msg string, args ...interface{}) bool {
	markHelper(a.t).Helper()
	return a.NotEqual(expected, actual, fmt.Sprintf(msg, args...))
}
