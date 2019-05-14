package golden_test

import (
	"gotest.tools/assert"
	"gotest.tools/golden"
	"gotest.tools/internal/examplet"
)

var t = examplet.T

func ExampleAssert() {
	actual := `
Sometimes there is a difference.

Sometimes there is not.
`
	golden.Assert(t, actual, "example-assert.golden")
}

func ExampleAssert_failure() {
	actual := `
Sometimes there is a difference.

This is one of those times.
`
	golden.Assert(t, actual, "example-assert.golden")
	// Output:
	// assertion failed:
	// --- expected
	// +++ actual
	// @@ -2,4 +2,4 @@
	//  Sometimes there is a difference.
	//
	// -Sometimes there is not.
	// +This is one of those times.
}

func ExampleAssert_failureInWhitespace() {
	actual := `
Sometimes  it  is just whitespace. 
`
	golden.Assert(t, actual, "example-assert-whitespace.golden")
	// Output:
	// assertion failed:
	// --- expected
	// +++ actual
	// @@ -1,3 +1,3 @@
	//
	// -Sometimes·it·is·just·whitespace.
	// +Sometimes··it··is·just·whitespace.·
}

func ExampleString() {
	assert.Assert(t, golden.String("foo", "foo-content.golden"))
}

func ExampleAssertBytes() {
	golden.AssertBytes(t, []byte("foo"), "foo-content.golden")
}
