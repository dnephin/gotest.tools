package testjson

import (
	"bytes"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/opt"
	"github.com/gotestyourself/gotestyourself/golden"
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
	assert.DeepEqual(t, exec, expectedExecution, cmpExecutionShallow)
}

var expectedExecution = &Execution{
	started: time.Now(),
	errors:  []string{"internal/broken/broken.go:5:21: undefined: somepackage"},
	packages: map[string]*Package{
		"github.com/gotestyourself/gotestyourself/testjson/internal/good": {
			run: 18,
			skipped: []TestCase{
				{Test: "TestSkipped"},
				{Test: "TestSkippedWitLog"},
			},
		},
		"github.com/gotestyourself/gotestyourself/testjson/internal/stub": {
			run: 28,
			failed: []TestCase{
				{Test: "TestFailed"},
				{Test: "TestFailedWithStderr"},
				{Test: "TestNestedWithFailure/c"},
				{Test: "TestNestedWithFailure"},
			},
			skipped: []TestCase{
				{Test: "TestSkipped"},
				{Test: "TestSkippedWitLog"},
			},
		},
	},
}

var cmpExecutionShallow = gocmp.Options{
	gocmp.AllowUnexported(Execution{}, Package{}),
	gocmp.FilterPath(stringPath("started"), opt.TimeWithThreshold(10*time.Second)),
	cmpPackageShallow,
}

var cmpPackageShallow = gocmp.Options{
	gocmp.FilterPath(opt.PathField(Package{}, "output"), gocmp.Ignore()),
	gocmp.Comparer(func(x, y TestCase) bool {
		return x.Test == y.Test
	}),
}

func stringPath(spec string) func(gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		return path.String() == spec
	}
}

func TestScanTestOutputWithDotsFormat(t *testing.T) {
	defer patchPkgPathPrefix("github.com/gotestyourself/gotestyourself")()

	shim := newConfigShim(dotsFormat, "go-test-json")
	exec, err := ScanTestOutput(shim.Config(t))

	assert.NilError(t, err)
	golden.Assert(t, shim.Out.String(), "dots-format.out")
	golden.Assert(t, shim.Err.String(), "dots-format.err")
	assert.DeepEqual(t, exec, expectedExecution, cmpExecutionShallow)
}
