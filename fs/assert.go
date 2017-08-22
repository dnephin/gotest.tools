package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"bytes"
	"sort"
	"strings"

	"github.com/stretchr/testify/require"
)

// Expected defines the expected state of a file or directory
type Expected interface {
	Mode() os.FileMode
}

type ExpectedFile interface {
	Expected
	Content() string
}

type ExpectedDir struct {
	mode    os.FileMode
	content map[string]Expected
}

func (d *ExpectedDir) Content() map[string]Expected {
	return d.content
}

func (d *ExpectedDir) Mode() os.FileMode {
	return d.mode
}

var _ Expected = &ExpectedDir{}

type ExpectOp func(Expected)

// ExpectDir creates an expectation that a directory exist at the path
func ExpectDir(ops ...ExpectOp) *ExpectedDir {
	dir := &ExpectedDir{}
	for _, op := range ops {
		op(dir)
	}
	return dir
}

func ExpectMode(mode os.FileMode) ExpectOp {
	return func(expected Expected) {
		switch typed := expected.(type) {
		case *ExpectedDir:
			typed.mode = mode
			// TODO: file
		}
	}
}

// None is a token which indicates the ExpectedFile should be empty
const None = "[NOTHING]"

const extra = "*"

const anyFileMode os.FileMode = 0

// Assert checks that the directory structure at path matches the expected tree
func Assert(t require.TestingT, root string, expected *ExpectedDir) bool {
	err := Compare(root, expected)
	switch err.(type) {
	case nil:
		return true
	case *compareError:
		t.Errorf(err.Error())
	default:
		require.NoError(t, err)
	}
	return false
}

// Compare the directory at path against the expected directory contents. Return
// an error if any files or directories do not match expected.
func Compare(root string, expected *ExpectedDir) error {
	result, err := compareRoot(root, expected)
	if err != nil {
		return err
	}

	failures, err := compareDir(root, expected)
	if err != nil {
		return err
	}

	failures.add(result)
	if failures.hasItems() {
		return failures.asError(root)
	}
	return nil
}

func compareRoot(path string, expected *ExpectedDir) (entryResult, error) {
	result := newEntryResult(path)

	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	compareMode(info, expected, &result)
	return result, nil
}

func compareDir(path string, expected *ExpectedDir) (failures, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return failures{}, err
	}
	content := expected.Content()
	extrasExpected, allowExtraFiles := content[extra]

	failures := failures{}
	for _, info := range fileInfos {
		fullPath := filepath.Join(path, info.Name())

		if expected, ok := content[info.Name()]; ok {
			results, err := compareDirEntry(fullPath, info, expected)
			if err != nil {
				return failures, err
			}
			failures.extend(results)
			continue
		}

		if !allowExtraFiles {
			failures.add(newEntryResult(fullPath, "unexpected file or directory"))
			continue
		}

		if extrasExpected != nil {
			result := newEntryResult(fullPath)
			compareMode(info, extrasExpected, &result)
			failures.add(result)
		}
	}

	// TODO: check for missing expected files/dirs
	return failures, nil
}

func compareDirEntry(fullPath string, info os.FileInfo, expected Expected) (failures, error) {
	result := newEntryResult(fullPath)
	compareMode(info, expected, &result)

	switch typed := expected.(type) {
	case ExpectedFile:
		if info.IsDir() {
			result.add("expected a file but found a directory")
		} else {
			// TODO: read file and compare to typed.Content() using difflib
		}
	case *ExpectedDir:
		if info.IsDir() {
			failures, err := compareDir(fullPath, typed)
			failures.add(result)
			return failures, err
		}
		result.add("expected a directory but found a file")
	}

	failures := failures{}
	return failures.add(result), nil
}

func compareMode(info os.FileInfo, expected Expected, result *entryResult) {
	if expected.Mode() != anyFileMode && expected.Mode() != info.Mode() {
		result.add("expected mode %s found %s", expected.Mode(), info.Mode())
	}
}

type failures struct {
	items []entryResult
}

func (f *failures) add(result entryResult) failures {
	if result.hasProblems() {
		f.items = append(f.items, result)
	}
	return *f
}

func (f *failures) extend(other failures) {
	f.items = append(f.items, other.items...)
}

func (f *failures) asError(root string) error {
	if f.hasItems() {
		return nil
	}

	sort.Slice(f.items, func(i, j int) bool {
		return f.items[i].path < f.items[j].path
	})

	buf := bytes.NewBufferString(root + "\n")
	for _, entry := range f.items {
		path, err := filepath.Rel(root, entry.path)
		if err != nil {
			return err
		}
		problems := strings.Join(entry.problems, ", ")
		buf.WriteString(fmt.Sprintf("%s: %s", path, problems))
	}

	return &compareError{formatted: buf.String()}
}

func (f *failures) hasItems() bool {
	return len(f.items) > 0
}

type entryResult struct {
	path     string
	problems []string
}

func newEntryResult(path string, problems ...string) entryResult {
	return entryResult{path: path, problems: problems}
}

func (f *entryResult) add(problem string, args ...interface{}) {
	f.problems = append(f.problems, fmt.Sprintf(problem, args...))
}

func (f *entryResult) hasProblems() bool {
	return len(f.problems) > 0
}

type compareError struct {
	formatted string
}

func (e *compareError) Error() string {
	return e.formatted
}
