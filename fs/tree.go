package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/stretchr/testify/require"
)

// Tree is a filesystem tree which has a directory as a root, and a list of files.
// Tree can be used to compare the structure and contents of directories.
type Tree struct {
	root  Dir
	Dirs  map[string]Tree
	Files []string
}

// NewTree returns a Tree populated from a filesystem path. No files or directories
// are created. NewTree will fail the test if the path does not exist.
func NewTree(t require.TestingT, path string) *Tree {
	trees, files := readTree(t, path)
	return &Tree{root: Dir{path: path}, Dirs: trees, Files: files}
}

func readTree(t require.TestingT, path string) (map[string]Tree, []string) {
	entries, err := ioutil.ReadDir(path)
	require.NoError(t, err)

	files := []string{}
	dirs := map[string]Tree{}
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			dirs[entry.Name()] = *NewTree(t, entryPath)
			continue
		}
		files = append(files, entry.Name())
	}
	return dirs, files
}

// Path returns the root path of the tree
func (tree *Tree) Path() string {
	return tree.root.Path()
}

func (tree *Tree) Assert(t require.TestingT, expected ExpectedTree) {

}

type ExpectedTree struct {
	files map[string]string
	dirs  map[string]*ExpectedTree
}

func NewExpectedTree() *ExpectedTree {
	return &ExpectedTree{
		files: make(map[string]string),
		dirs:  make(map[string]*ExpectedTree),
	}
}

// AddFile adds a file that is expcted to be in a tree. The filename should be
// a relative path to the root of the tree. Contents is the content expected
// in the file. The empty string will match any content.
func (e *ExpectedTree) AddFile(path, content string) {
	dir, filename := filepath.Split(path)
	switch dir {
	case "./", "", "/":
		e.files[filename] = content
		return
	}

	parts := strings.Split(dir, string(os.PathSeparator))
	tree := e
	for _, part := range parts {
		tree = tree.dir(part)
	}
	tree.files[filename] = content
}

func (e *ExpectedTree) AddDir(dir string) {

}

func (e *ExpectedTree) dir(name string) *ExpectedTree {
	_, ok := e.dirs[name]
	if !ok {
		e.dirs[name] = NewExpectedTree()
	}
	return e.dirs[name]
}
