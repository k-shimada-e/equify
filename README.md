# equify

**Testify-style assertions that respect your types' `Equal` methods.**

English | [日本語](README.ja.md)

equify combines the familiar API of [testify](https://github.com/stretchr/testify) with the comparison rules of [go-cmp](https://github.com/google/go-cmp). Use `assert.Equal(t, want, got)` as usual, and let a type's `Equal` method define equality—even when that type is nested inside a struct, slice, or map.

- Package functions, `assert.New(t)` methods, and `require` assertions.
- Custom messages and formatted variants such as `Equalf`.
- Testify-style failure summaries with go-cmp diffs.
- Comparison options for ignoring fields, equating empty collections, and more.
- Other assertions delegate to testify.

## Installation

Requires **Go 1.23 or later**.

```sh
go get github.com/k-shimada-e/equify@v0.1.1
```

Import the root package as `assert` and the `require` subpackage as shown below.

## Quick start

Two `time.Time` values can represent the same instant with different internal representations. equify uses `Time.Equal` to compare them:

```go
package example_test

import (
	"testing"
	"time"

	assert "github.com/k-shimada-e/equify"
	"github.com/k-shimada-e/equify/require"
)

func TestInstant(t *testing.T) {
	want := time.Unix(100, 0).UTC()
	got := want.In(time.FixedZone("offset", 3600))

	assert.Equal(t, want, got)
	assert.Equalf(t, want, got, "event %d", 42)
	assert.New(t).Equal(want, got)

	// Stop the test immediately if the assertion fails.
	require.Equal(t, want, got)
	require.New(t).Equal(want, got)
}
```

`assert.Equal` and `assert.NotEqual` return a `bool` and allow the test to continue after failure. Their `require` counterparts have no return value and call `FailNow` on failure. Call `require` assertions from the goroutine running the test.

## Define equality for your own types

When a type has a supported `Equal` method, its result determines equality. In this example, users are equal when their IDs match:

```go
package example_test

import (
	"testing"

	assert "github.com/k-shimada-e/equify"
)

type User struct {
	ID   int
	Name string
}

func (u User) Equal(other User) bool {
	return u.ID == other.ID
}

func TestUsers(t *testing.T) {
	want := User{ID: 1, Name: "Alice"}
	got := User{ID: 1, Name: "Alicia"}

	assert.Equal(t, want, got) // Passes: the IDs match.
	assert.Equal(t, []User{want}, []User{got}) // Elements use User.Equal too.
}
```

Method signatures must follow [go-cmp's rules](https://pkg.go.dev/github.com/google/go-cmp/cmp#Equal), such as `Equal(T) bool` or `Equal(interface{}) bool`. Arbitrarily named comparison methods are not detected. Your `Equal` method must be deterministic and symmetric.

## Failure output

Comparing users with different IDs produces output like this. Paths, line numbers, and whitespace may vary:

```text
Error Trace: example_test.go:42
Error:       Not equal:
             expected: example_test.User{ID:1, Name:"Alice"}
             actual  : example_test.User{ID:2, Name:"Bob"}

             Diff:
             --- Expected
             +++ Actual
               example_test.User(
             -   {ID: 1, Name: "Alice"},
             +   {ID: 2, Name: "Bob"},
               )
Messages:    user comparison
```

The `Messages` entry appears when you supply a message, for example `assert.Equal(t, want, got, "user comparison")`.

The `Not equal:` summary, `expected`/`actual` values, and `Diff:` headings follow testify's layout. Diff lines marked `-` show the expected value; `+` shows the actual value. The diff body uses go-cmp's format.

An `Equal` method returns only a `bool`, so it cannot explain why a comparison failed. go-cmp displays the entire value at that comparison point, including fields the method does not compare. Types compared recursively without an `Equal` method can produce field-level diffs.

## Customize comparisons

Keep existing `Equal` signatures and pass options through `NewWithOptions` or the explicit options functions:

```go
package example_test

import (
	"testing"

	assert "github.com/k-shimada-e/equify"
	"github.com/k-shimada-e/equify/require"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type Record struct {
	ID    int
	Cache string
}

func TestComparisonOptions(t *testing.T) {
	// Apply options to this instance's equality assertions.
	a := assert.NewWithOptions(t, cmpopts.EquateEmpty())
	a.Equal([]int(nil), []int{})

	// Apply options to one comparison.
	want := Record{ID: 1, Cache: "old"}
	got := Record{ID: 1, Cache: "new"}
	assert.EqualWithOptions(t, want, got,
		[]cmp.Option{cmpopts.IgnoreFields(Record{}, "Cache")})

	// Configure a require instance in the same way.
	r := require.NewWithOptions(t, cmpopts.EquateEmpty())
	r.Equal([]int(nil), []int{})
}
```

`assert.NotEqualWithOptions` is also available. Instance options apply to `Equal`, `Equalf`, `NotEqual`, and `NotEqualf`; they do not affect delegated methods such as `EqualValues`.

## Migrate from testify

Replace the imports and keep your assertion calls:

```diff
- "github.com/stretchr/testify/assert"
- "github.com/stretchr/testify/require"
+ assert "github.com/k-shimada-e/equify"
+ "github.com/k-shimada-e/equify/require"
```

Run your tests after migrating: the API is familiar, but equality semantics change.

### APIs with new comparison behavior

| API | Behavior |
| --- | --- |
| `Equal`, `Equalf` | Compare with go-cmp; show a diff on failure |
| `NotEqual`, `NotEqualf` | Use the same comparison rules; fail when values are equal |
| `assert.ObjectsAreEqual` | Use the same comparison rules; return a `bool` |
| Other public APIs | Delegate to testify |

Changes to the `Equal`/`NotEqual` family apply to package functions, assertion methods, and `require` variants.

`EqualValues`, `NotEqualValues`, `EqualExportedValues`, `Exactly`, `ElementsMatch`, and `ObjectsAreEqualValues` keep testify's internal comparisons. equify does not replace every comparison in testify with go-cmp.

### Compatibility scope

This initial implementation targets **testify v1.11.1** package function signatures and the assertion methods available through `New(t)`.

- equify's `*Assertions` is a different type from upstream's `*Assertions`. Code accepting the upstream concrete type needs adaptation.
- Assertions created internally by `testify/suite` still use testify. Call equify functions or create a equify instance explicitly to use its comparisons.
- This library does not provide replacements for `testify/mock` or `testify/suite`.
- Failure output follows testify's layout, but diff bodies are not byte-for-byte identical.

### Default comparison rules

- Supported `Equal` methods define equality, including for nested values.
- Structs without those methods include unexported fields in comparisons. Unlike plain go-cmp, equify enables access to those fields with `cmp.Exporter`.
- `nil` and empty slices remain different. `int` and `int64` remain different.
- Top-level function arguments to `Equal`/`NotEqual` fail the assertion, as they do in testify.
- Panics from invalid comparison options or comparison methods propagate to the caller.

Other behavior, including cyclic value comparisons, follows go-cmp. There is no fallback to testify's `reflect.DeepEqual` when options are omitted.

## Development

Dependency versions are pinned in `go.mod` and `go.sum`.

```sh
go mod download
go test ./...
go vet ./...
```

`forward.go` and `require/forward.go` are generated wrappers around the pinned testify version. Do not edit them manually. After changing that dependency, regenerate and validate:

```sh
go generate ./...
go test ./...
go vet ./...
```

Tests cover custom and nested `Equal` methods, `time.Time`, comparison options, testify-style failure summaries, caller locations, and `require` stopping on failure.

## License

[MIT License](LICENSE), the same license used by testify. Failure-summary formatting is adapted from testify; its copyright notice is retained in the license file. Dependencies retain their respective licenses.
