package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareMissingRoot(t *testing.T) {
	err := Compare("/bogus/path/does/not/exist", ExpectDir())
	expected := "stat /bogus/path/does/not/exist: no such file or directory"
	assert.EqualError(t, err, expected)
}

func TestCompareRootModeMismatch(t *testing.T) {
	// TODO: use withMode
	dir := NewDir(t, "assert-test-root", )
	defer dir.Remove()
	err := Compare(dir.Path(), ExpectDir(ExpectMode(0700)))
	assert.EqualError(t, err, "foo")
}

func TestCompareRootTypeMismatch(t *testing.T) {

}

func TestCompareRootSuccess(t *testing.T) {

}
