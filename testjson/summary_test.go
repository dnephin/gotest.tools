package testjson

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/jonboulle/clockwork"
)

func TestPrintSummaryNoFailures(t *testing.T) {
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
	err := PrintSummary(out, exec)
	assert.NilError(t, err)

	expected := "\nDONE 13 tests in 34.123s\n"
	assert.Equal(t, out.String(), expected)
}

func TestPrintSummaryWithFailures(t *testing.T) {
	defer patchPkgPathPrefix("example.com")()
	fake, reset := patchClock()
	defer reset()

	out := new(bytes.Buffer)
	exec := &Execution{
		started: fake.Now(),
		packages: map[string]*Package{
			"example.com/project/fs": {
				run: 12,
				failed: []TestCase{
					{
						Package: "example.com/project/fs",
						Test:    "TestFileDo",
						Elapsed: 1411 * time.Millisecond,
					},
					{
						Package: "example.com/project/fs",
						Test:    "TestFileDoError",
						Elapsed: 12 * time.Millisecond,
					},
				},
				output: map[string][]string{
					"TestFileDo": multiLine(`=== RUN   TestFileDo
Some stdout/stderr here
--- FAIL: TestFailDo (1.41s)
	do_test.go:33 assertion failed
`),
					"TestFileDoError": multiLine(`=== RUN   TestFileDoError
--- FAIL: TestFailDoError (0.01s)
	do_test.go:50 assertion failed: expected nil error, got WHAT!
`),
				},
			},
			"example.com/project/pkg/more": {
				run: 1,
				failed: []TestCase{
					{
						Package: "example.com/project/pkg/more",
						Test:    "TestAlbatross",
						Elapsed: 40 * time.Millisecond,
					},
				},
				skipped: []TestCase{
					{
						Package: "example.com/project/pkg/more",
						Test:    "TestOnlySometimes",
						Elapsed: 0,
					},
				},
				output: map[string][]string{
					"TestAlbatross": multiLine(`=== RUN   TestAlbatross
--- FAIL: TestAlbatross (0.04s)
`),
					"TestOnlySometimes": multiLine(`=== RUN   TestOnlySometimes
--- SKIP: TestOnlySometimes (0.00s)
	good_test.go:27: the skip message
`),
				},
			},
			"example.com/project/badmain": {
				action: ActionFail,
				output: map[string][]string{
					"": {"sometimes main can exit 2\n"},
				},
			},
		},
		errors: []string{
			"pkg/file.go:99:12: missing ',' before newline",
		},
	}
	fake.Advance(34123111 * time.Microsecond)
	err := PrintSummary(out, exec)
	assert.NilError(t, err)

	expected := `
DONE 13 tests, 1 skipped, 4 failures, 1 error in 34.123s

=== Skipped
=== SKIP: project/pkg/more TestOnlySometimes (0.00s)
	good_test.go:27: the skip message


=== Failures
=== FAIL: project/badmain  (0.00s)
sometimes main can exit 2

=== FAIL: project/fs TestFileDo (1.41s)
Some stdout/stderr here
	do_test.go:33 assertion failed

=== FAIL: project/fs TestFileDoError (0.01s)
	do_test.go:50 assertion failed: expected nil error, got WHAT!

=== FAIL: project/pkg/more TestAlbatross (0.04s)


=== Errors
pkg/file.go:99:12: missing ',' before newline
`
	assert.Equal(t, out.String(), expected)
}

func patchClock() (clockwork.FakeClock, func()) {
	fake := clockwork.NewFakeClock()
	clock = fake
	return fake, func() { clock = clockwork.NewRealClock() }
}

func multiLine(s string) []string {
	return strings.SplitAfter(s, "\n")
}
