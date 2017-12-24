package main

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestRun(t *testing.T) {
	dir := fs.NewDir(t, "test-run", fs.FromDir("testdata/full"))
	log.SetFlags(0)
	err := run(&options{
		dirs: []string{dir.Path()},
	})
	assert.NilError(t, err)

	raw, err := ioutil.ReadFile(dir.Join("some_test.go"))
	assert.NilError(t, err)
	golden.Assert(t, string(raw), "full-expected/some_test.go")
}
