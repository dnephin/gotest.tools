package examplet // import "gotest.tools/internal/examplet"

import (
	"fmt"
)

// T provides an implementation of assert.TestingT which fmt.Println. This
// implementation can be used in godoc examples to print expected output
// without failing the example.
var T = t{}

type t struct{}

// FailNow does nothing.
func (t t) FailNow() {
}

// Fail does nothing.
func (t t) Fail() {}

// Log args by printing them to stdout
func (t t) Log(args ...interface{}) {
	fmt.Println(args...)
}
