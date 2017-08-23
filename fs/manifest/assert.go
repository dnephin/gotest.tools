package manifest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/pmezard/go-difflib/difflib"
)

// Expected defines the expected state of a file or directory
type Expected interface {
	Mode() os.FileMode
}

// ExpectedDir defines the expected state of a directory that can be compared
// Compare() or Assert()
type ExpectedDir interface {
	Expected
	Content() map[string]Expected
}

type expectedFile struct {
	mode    os.FileMode
	content string
}

func (f *expectedFile) Content() string {
	return f.content
}

func (f *expectedFile) Mode() os.FileMode {
	return f.mode
}

type expectedDir struct {
	mode    os.FileMode
	content map[string]Expected
}

func (d *expectedDir) Content() map[string]Expected {
	return d.content
}

func (d *expectedDir) Mode() os.FileMode {
	return d.mode
}

func newExpectedDir() *expectedDir {
	return &expectedDir{content: make(map[string]Expected)}
}

var _ Expected = &expectedDir{}

// ExpectOp is an operation that can modify the expected file or directory
type ExpectOp func(Expected)

// ExpectDir creates an expectation that a directory exist at the path
func ExpectDir(ops ...ExpectOp) ExpectedDir {
	dir := newExpectedDir()
	for _, op := range ops {
		op(dir)
	}
	return dir
}

// ExpectSubDir creates an expectation that a directory exists within another
// directory
func ExpectSubDir(name string, ops ...ExpectOp) ExpectOp {
	return func(expected Expected) {
		switch typed := expected.(type) {
		case *expectedDir:
			expected := newExpectedDir()
			typed.content[name] = expected
			applyExpectOps(expected, ops)
		default:
			panic(fmt.Sprintf(
				"ExpectSubDir can not operate on a %T. Use ExpectDir()", typed))
		}
	}
}

// ExpectMode creates an expectation that a file or directory has a specific file
// mode
func ExpectMode(mode os.FileMode) ExpectOp {
	return func(expected Expected) {
		switch typed := expected.(type) {
		case *expectedDir:
			typed.mode = mode
		case *expectedFile:
			typed.mode = mode
		}
	}
}

// AllowExtras allows other files to exist in a directory, without having to
// enumerating every file.
func AllowExtras() ExpectOp {
	return func(expected Expected) {
		switch typed := expected.(type) {
		case *expectedDir:
			typed.content[ExtraFiles] = nil
		default:
			panic(fmt.Sprintf(
				"AllowExtras can not operate on a %T. Use ExpectDir()", typed))
		}
	}
}

// ExpectFile creates an expectation that a directory contains a specific file
func ExpectFile(name string, ops ...ExpectOp) ExpectOp {
	return func(expected Expected) {
		switch typed := expected.(type) {
		case *expectedDir:
			expected := &expectedFile{}
			typed.content[name] = expected
			applyExpectOps(expected, ops)
		default:
			panic(fmt.Sprintf(
				"ExpectedFile can not operate on a %T. Use ExpectDir()", typed))
		}
	}
}

// ExpectContent creates an expectation that a file contains specific content
func ExpectContent(content string) ExpectOp {
	return func(expected Expected) {
		switch typed := expected.(type) {
		case *expectedFile:
			typed.content = content
		default:
			panic(fmt.Sprintf(
				"ExpectedContent can not operate on a %T. Use ExpectDir()", typed))
		}
	}
}

func applyExpectOps(expected Expected, ops []ExpectOp) {
	for _, op := range ops {
		op(expected)
	}
}

// Empty is a token which indicates the expected file should be empty
const Empty = "[NOTHING]"

// ExtraFiles is a special token to match against any file or directory in a
// directory
const ExtraFiles = "*"

const anyFileMode os.FileMode = 0

// Assert checks that the directory structure at path matches the expected tree
// by calling Compare and failing the test if there is an error
func Assert(t assert.TestingT, root string, expected ExpectedDir) bool {
	err := Compare(root, expected)
	switch err.(type) {
	case nil:
		return true
	case *compareError:
		t.Log(err.Error())
		t.Fail()
	default:
		assert.NilError(t, err)
	}
	return false
}

// Compare the directory at path against the expected directory contents. Return
// an error if any files or directories do not match expected.
func Compare(root string, expected ExpectedDir) error {
	result, isDir, err := compareRoot(root, expected)
	if err != nil {
		return err
	}

	if !isDir {
		var failures failures
		failures.add(result)
		return failures.asError(root)
	}

	failures, err := compareDir(root, expected)
	if err != nil {
		return err
	}

	failures.add(result)
	return failures.asError(root)
}

func compareRoot(path string, expected Expected) (entryResult, bool, error) {
	result := newEntryResult(path)

	info, err := os.Stat(path)
	if err != nil {
		return result, false, err
	}
	compareMode(info, expected, &result)
	if !info.IsDir() {
		result.add("expected a directory but found a file")
	}
	return result, info.IsDir(), nil
}

func compareDir(path string, expected ExpectedDir) (failures, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return failures{}, err
	}
	content := expected.Content()
	extrasExpected, allowExtraFiles := content[ExtraFiles]

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

	missing := findMissing(path, expected, fileInfos)
	failures.extend(missing)
	return failures, nil
}

func compareDirEntry(fullPath string, info os.FileInfo, expected Expected) (failures, error) {
	result := newEntryResult(fullPath)
	compareMode(info, expected, &result)

	switch typed := expected.(type) {
	case *expectedFile:
		if info.IsDir() {
			result.add("expected a file but found a directory")
		} else {
			err := compareFileContent(fullPath, typed, &result)
			if err != nil {
				return failures{}, err
			}
		}
	case *expectedDir:
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

func compareFileContent(path string, expected *expectedFile, result *entryResult) error {
	if expected.Content() == "" {
		return nil
	}
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if expected.Content() == Empty && len(raw) != 0 {
		result.add("expected an empty file, but got %d bytes", len(raw))
		return nil
	}

	content := string(raw)
	if expected.Content() != content {
		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expected.Content()),
			B:        difflib.SplitLines(content),
			FromFile: "Expected",
			ToFile:   "Actual",
			Context:  3,
		})
		if err != nil {
			return err
		}
		result.add("file content was not as expected:\n%s", indentDiff(diff))
	}
	return nil
}

func indentDiff(diff string) string {
	return "   " + strings.Replace(strings.TrimSpace(diff), "\n", "\n   ", -1) + "\n"
}

// nolint: interfacer
func compareMode(info os.FileInfo, expected Expected, result *entryResult) {
	if expected.Mode() != anyFileMode && expected.Mode() != info.Mode() {
		result.add("expected mode %s got %s", expected.Mode(), info.Mode())
	}
}

func findMissing(path string, expected ExpectedDir, infos []os.FileInfo) failures {
	paths := map[string]struct{}{}
	for _, info := range infos {
		paths[info.Name()] = struct{}{}
	}
	failures := failures{}
	for expectedName := range expected.Content() {
		if expectedName == ExtraFiles {
			continue
		}
		if _, exists := paths[expectedName]; !exists {
			fullPath := filepath.Join(path, expectedName)
			failures.add(newEntryResult(fullPath, "does not exist"))
		}
	}
	return failures
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
	if !f.hasItems() {
		return nil
	}

	sort.Slice(f.items, func(i, j int) bool {
		return f.items[i].path < f.items[j].path
	})

	buf := bytes.NewBufferString(root)
	for _, entry := range f.items {
		path, err := filepath.Rel(root, entry.path)
		if err != nil {
			return err
		}
		problems := strings.Join(entry.problems, ", ")
		buf.WriteString(fmt.Sprintf("\n%s: %s", path, problems))
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
