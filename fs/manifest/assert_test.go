package manifest

import (
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
)

func TestCompareMissingRoot(t *testing.T) {
	err := Compare("/bogus/path/does/not/exist", ExpectDir())
	expected := "stat /bogus/path/does/not/exist: no such file or directory"
	assert.Error(t, err, expected)
}

func TestCompareRootModeMismatch(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root", fs.WithMode(0500))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(ExpectMode(0700|os.ModeDir)))
	expected := dir.Path() + "\n.: expected mode drwx------ got dr-x------"
	assert.Error(t, err, expected)
}

func TestCompareRootTypeMismatch(t *testing.T) {
	file := fs.NewFile(t, "assert-test-root")
	defer file.Remove()

	err := Compare(file.Path(), ExpectDir())
	expected := file.Path() + "\n.: expected a directory but found a file"
	assert.Error(t, err, expected)
}

func TestCompareRootSuccess(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root", fs.WithMode(0700))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(ExpectMode(0700|os.ModeDir)))
	assert.NilError(t, err)
}

func TestCompareDirectoryWithExtras(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root", fs.WithMode(0700))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(ExpectMode(0700|os.ModeDir)))
	assert.NilError(t, err)
}

func TestCompareDirectoryWithAllowedExtras(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithFile("extra", "some content"))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir())
	expected := dir.Path() + "\nextra: unexpected file or directory"
	assert.Error(t, err, expected)
}

func TestCompareRootDirectoryWithExtrasExpectedMismatch(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithFile("extra", "some content", fs.WithMode(0600)))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(
		ExpectFile(ExtraFiles, ExpectMode(0555))))
	expected := dir.Path() + "\nextra: expected mode -r-xr-xr-x got -rw-------"
	assert.Error(t, err, expected)
}

func TestCompareRootDirectoryWithExtrasExpected(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithFile("extra", "some content"))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(AllowExtras()))
	assert.NilError(t, err)
}

func TestCompareMissingExpectedFile(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root")
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(
		ExpectFile("file1"),
		ExpectSubDir("dir1", ExpectFile("file2")),
	))
	expected := dir.Path() + "\ndir1: does not exist\nfile1: does not exist"
	assert.Error(t, err, expected)
}

func TestCompareSubDirectoryFileMismatch(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithDir("dir1", fs.WithFile("file1", "content")))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(
		ExpectSubDir("dir1", ExpectFile("file2")),
	))
	expected := dir.Path() + `
dir1/file1: unexpected file or directory
dir1/file2: does not exist`
	assert.Error(t, err, expected)
}

func TestCompareFileContentNotEmpty(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithFile("file1", "not empty\n"))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(
		ExpectFile("file1", ExpectContent(Empty))))
	expected := dir.Path() + "\nfile1: expected an empty file, but got 10 bytes"
	assert.Error(t, err, expected)
}

func TestCompareFileContentMatches(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithFile("file1", "not empty\n"))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(
		ExpectFile("file1", ExpectContent("not empty\n"))))
	assert.NilError(t, err)
}

func TestCompareFileContentMisMatch(t *testing.T) {
	dir := fs.NewDir(t, "assert-test-root",
		fs.WithFile("file1", "line1\nline2\nline3\n"))
	defer dir.Remove()

	err := Compare(dir.Path(), ExpectDir(
		ExpectFile("file1", ExpectContent("line2\nline3\n"))))
	expected := dir.Path() + `
file1: file content was not as expected:
   --- Expected
   +++ Actual
   @@ -1,3 +1,4 @@
   +line1
    line2
    line3
`
	assert.Error(t, err, expected)
}
