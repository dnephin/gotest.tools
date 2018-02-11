package testjson

import (
	"bytes"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
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
