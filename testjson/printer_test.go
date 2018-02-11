package testjson

import (
	"bytes"
	"testing"
	"time"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/jonboulle/clockwork"
)

func TestRelativePackagePath(t *testing.T) {
	relPath := relativePackagePath(
		"github.com/gotestyourself/gotestyourself/testjson/extra/relpath")
	assert.Equal(t, relPath, "extra/relpath")

	relPath = relativePackagePath(
		"github.com/gotestyourself/gotestyourself/testjson")
	assert.Equal(t, relPath, ".")
}

func TestGetPkgPathPrefix(t *testing.T) {
	assert.Equal(t, pkgPathPrefix, "github.com/gotestyourself/gotestyourself/testjson")
}

func TestCondensedFormat(t *testing.T) {
	defer patchPkgPathPrefix("github.com/gotestyourself/gotestyourself")()
	goTestOutput := golden.Get(t, "go-test-json-output")
	out := new(bytes.Buffer)
	_, err := ScanTestOutput(bytes.NewReader(goTestOutput), out, condensedFormat)
	assert.NilError(t, err)

	golden.Assert(t, out.String(), "condensed-format")
}

func patchPkgPathPrefix(val string) func() {
	var oldVal string
	oldVal, pkgPathPrefix = pkgPathPrefix, val
	return func() { pkgPathPrefix = oldVal }
}

func TestDotsFormat(t *testing.T) {
	defer patchPkgPathPrefix("github.com/gotestyourself/gotestyourself")()
	goTestOutput := golden.Get(t, "go-test-json-output")
	out := new(bytes.Buffer)
	_, err := ScanTestOutput(bytes.NewReader(goTestOutput), out, dotsFormat)
	assert.NilError(t, err)

	golden.Assert(t, out.String(), "dots-format")
}

func TestPrintExecutionNoFailures(t *testing.T) {
	fake, reset := patchClock()
	defer reset()

	out := new(bytes.Buffer)
	exec := &Execution{
		started: fake.Now(),
		packages: map[string]*Package{
			"foo":   {run: 12},
			"other": {run: 1},
		},
	}
	fake.Advance(34123111 * time.Microsecond)
	err := PrintExecution(out, exec)
	assert.NilError(t, err)

	expected := "\nDONE 13 tests in 34.123s\n"
	assert.Equal(t, out.String(), expected)
}

func TestPrintExecutionWithFailures(t *testing.T) {
	fake, reset := patchClock()
	defer reset()

	out := new(bytes.Buffer)
	exec := &Execution{
		started: fake.Now(),
		packages: map[string]*Package{
			"example.com/project/fs": {
				run: 12,
				failed: []TestEvent{
					{
						Package: "example.com/project/fs",
						Test:    "TestFileDo",
						Output:  "something",
						Elapsed: 1.1411,
					},
					{
						Package: "example.com/project/fs",
						Test:    "TestFileDoError",
						Output:  "something",
						Elapsed: 0.12,
					},
				},
				output: map[string]*bytes.Buffer{
					"TestFileDo": bytes.NewBufferString(`=== RUN   TestFileDo
    do_test.go:33 assertion failed
--- FAIL: TestFailDo (1.41s)
`),
					"TestFileDoError": bytes.NewBufferString(""),
				},
			},
			"example.com/project/pkg/more": {
				run: 1,
				failed: []TestEvent{
					{
						Package: "example.com/project/pkg/more",
						Test:    "TestAlbatross",
						Output:  "something",
						Elapsed: 0,
					},
				},
				output: map[string]*bytes.Buffer{
					"TestAlbatross": bytes.NewBufferString(""),
				},
			},
		},
	}
	fake.Advance(34123111 * time.Microsecond)
	err := PrintExecution(out, exec)
	assert.NilError(t, err)

	expected := `
DONE 13 tests with 3 failure(s) in 34.123s
=== RUN   TestFileDo
    do_test.go:33 assertion failed
--- FAIL: TestFailDo (1.41s)
// TODO: add other output
`
	assert.Equal(t, out.String(), expected)
}

func patchClock() (clockwork.FakeClock, func()) {
	fake := clockwork.NewFakeClock()
	clock = fake
	return fake, func() { clock = clockwork.NewRealClock() }
}
