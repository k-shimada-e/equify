package assert_test

import (
	"testing"

	"github.com/k-shimada-e/equify/assert"
	testify "github.com/stretchr/testify/assert"
)

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
