package testjson

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
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
