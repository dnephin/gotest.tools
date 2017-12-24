package main

import (
	"bytes"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
	"go/format"
	"go/parser"
	"go/token"
	"testing"
)

func TestReplaceTestingT(t *testing.T) {
	source := `
package foo

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
	a := assert.TestingT
	b := require.TestingT
	c := require.TestingT(t)
	if a == b {}
}

func do(t require.TestingT) {}
`
	migration := newMigrationFromSource(t, source)
	migrateFile(migration)

	expected := `package foo

import (
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
	"testing"
)

func TestSomething(t *testing.T) {
	a := assert.TestingT
	b := assert.TestingT
	c := assert.TestingT(t)
	if a == b {
	}
}

func do(t assert.TestingT) {}
`

	buf := new(bytes.Buffer)
	format.Node(buf, migration.fileset, migration.file)
	assert.Assert(t, cmp.EqualMultiLine(expected, buf.String()))
}

func newMigrationFromSource(t *testing.T, source string) migration {
	fileset := token.NewFileSet()
	nodes, err := parser.ParseFile(
		fileset,
		"foo.go",
		source,
		parser.AllErrors|parser.ParseComments)
	assert.NilError(t, err)
	return migration{
		file:        nodes,
		fileset:     fileset,
		importNames: newImportNames(nodes.Imports),
	}
}
