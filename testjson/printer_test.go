package testjson

import (
	"bytes"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/opt"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/jonboulle/clockwork"
)

//go:generate ./generate.sh

type scanConfigShim struct {
	inputName string
	handler   HandleEvent
	Out       *bytes.Buffer
	Err       *bytes.Buffer
}

func (s *scanConfigShim) Config(t *testing.T) ScanConfig {
	return ScanConfig{
		Stdout:  bytes.NewReader(golden.Get(t, s.inputName+".out")),
		Stderr:  bytes.NewReader(golden.Get(t, s.inputName+".err")),
		Out:     s.Out,
		Err:     s.Err,
		Handler: s.handler,
	}
}

func newConfigShim(handler HandleEvent, inputName string) *scanConfigShim {
	return &scanConfigShim{
		inputName: inputName,
		handler:   handler,
		Out:       new(bytes.Buffer),
		Err:       new(bytes.Buffer),
	}
}

func patchPkgPathPrefix(val string) func() {
	var oldVal string
	oldVal, pkgPathPrefix = pkgPathPrefix, val
	return func() { pkgPathPrefix = oldVal }
}

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

func TestScanTestOutputWithShortVerboseFormat(t *testing.T) {
	defer patchPkgPathPrefix("github.com/gotestyourself/gotestyourself")()

	shim := newConfigShim(shortVerboseFormat, "go-test-json")
	exec, err := ScanTestOutput(shim.Config(t))

	assert.NilError(t, err)
	golden.Assert(t, shim.Out.String(), "short-verbose-format.out")
	golden.Assert(t, shim.Err.String(), "short-verbose-format.err")
	expected := &Execution{
		started: time.Now(),
		errors:  []string{"internal/broken/broken.go:5:21: undefined: somepackage"},
	}
	assert.DeepEqual(t, exec, expected, cmpExecutionShalow)
}

var cmpExecutionShalow = gocmp.Options{
	gocmp.AllowUnexported(Execution{}, Package{}),
	gocmp.FilterPath(stringPath("started"), opt.TimeWithThreshold(10*time.Second)),
	// TODO: remove ignore
	gocmp.FilterPath(stringPath("packages"), gocmp.Ignore()),
}

//var cmpPackage = cmpopts.IgnoreFields(Package{}, "output")

func stringPath(spec string) func(gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		return path.String() == spec
	}
}

func TestScanTestOutputWithDotsFormat(t *testing.T) {
	defer patchPkgPathPrefix("github.com/gotestyourself/gotestyourself")()

	shim := newConfigShim(dotsFormat, "go-test-json")
	_, err := ScanTestOutput(shim.Config(t))

	assert.NilError(t, err)
	golden.Assert(t, shim.Out.String(), "dots-format.out")
	golden.Assert(t, shim.Err.String(), "dots-format.err")
	// TODO: compare Execution
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
		errors: []string{
			"pkg/file.go:99:12: missing ',' before newline",
		},
	}
	fake.Advance(34123111 * time.Microsecond)
	err := PrintExecution(out, exec)
	assert.NilError(t, err)

	expected := `
DONE 13 tests, 3 failures, 1 error in 34.123s
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
