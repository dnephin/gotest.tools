package main

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
	"golang.org/x/tools/go/loader"
)

func TestMigrateFileReplacesTestingT(t *testing.T) {
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
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
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
	actual, err := formatFile(migration)
	assert.NilError(t, err)
	assert.Assert(t, cmp.EqualMultiLine(expected, string(actual)))
}

func newMigrationFromSource(t *testing.T, source string) migration {
	fileset := token.NewFileSet()
	nodes, err := parser.ParseFile(
		fileset,
		"foo.go",
		source,
		parser.AllErrors|parser.ParseComments)
	assert.NilError(t, err)

	opts := options{}
	conf := loader.Config{
		Fset:        fileset,
		ParserMode:  parser.ParseComments,
		Build:       buildContext(opts),
		AllowErrors: true,
	}
	conf.TypeChecker.Error = func(e error) {}
	conf.CreateFromFiles("foo.go", nodes)
	prog, err := conf.Load()
	assert.NilError(t, err)

	pkgInfo := prog.InitialPackages()[0]

	return migration{
		file:        pkgInfo.Files[0],
		fileset:     fileset,
		importNames: newImportNames(nodes.Imports, opts),
		pkgInfo:     pkgInfo,
	}
}

func TestMigrateFileWithNamedCmpPackage(t *testing.T) {
	source := `
package foo

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSomething(t *testing.T) {
	assert.Equal(t, "a", "b")
}
`
	migration := newMigrationFromSource(t, source)
	migration.importNames.cmp = "is"
	migrateFile(migration)

	expected := `package foo

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestSomething(t *testing.T) {
	assert.Check(t, is.Equal("a", "b"))
}
`
	actual, err := formatFile(migration)
	assert.NilError(t, err)
	assert.Assert(t, cmp.EqualMultiLine(expected, string(actual)))
}

func TestMigrateFileWithCommentsOnAssert(t *testing.T) {
	source := `
package foo

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSomething(t *testing.T) {
	// This is going to fail
	assert.Equal(t, "a", "b")
}
`
	migration := newMigrationFromSource(t, source)
	migrateFile(migration)

	expected := `package foo

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestSomething(t *testing.T) {
	// This is going to fail
	assert.Check(t, cmp.Equal("a", "b"))
}
`
	actual, err := formatFile(migration)
	assert.NilError(t, err)
	assert.Assert(t, cmp.EqualMultiLine(expected, string(actual)))
}

func TestMigrateFileConvertNilToNilError(t *testing.T) {
	source := `
package foo

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestSomething(t *testing.T) {
	var err error
	assert.Nil(t, err)
	require.Nil(t, err)
}
`
	migration := newMigrationFromSource(t, source)
	migrateFile(migration)

	expected := `package foo

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestSomething(t *testing.T) {
	var err error
	assert.Check(t, cmp.NilError(err))
	assert.NilError(t, err)
}
`
	actual, err := formatFile(migration)
	assert.NilError(t, err)
	assert.Assert(t, cmp.EqualMultiLine(expected, string(actual)))
}

func TestMigrateFileConvertAssertNew(t *testing.T) {
	source := `
package foo

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
	assert := require.New(t)

	assert.Equal("one", "two")
	assert.NotEqual("one", "two")

	is := assert.New(t)
	is.Equal("one", "two")
	is.NotEqual("one", "two")
}
`
	migration := newMigrationFromSource(t, source)
	migrateFile(migration)

	expected := `package foo

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestSomething(t *testing.T) {

	assert.Assert(t, cmp.Equal("one", "two"))
	assert.Assert(t, "one" != "two")

	assert.Check(t, cmp.Equal("one", "two"))
	assert.Check(t, "one" != "two")
}
`
	actual, err := formatFile(migration)
	assert.NilError(t, err)
	assert.Assert(t, cmp.EqualMultiLine(expected, string(actual)))
}

func TestMigrateFileConvertNotTestingT(t *testing.T) {
	source := `
package foo

import (
	"testing"

	"github.com/go-check/check"
	"github.com/stretchr/testify/assert"
)

func TestWithChecker(c *check.C) {
	var err error
	assert.NilError(c, err)
}

func HelperWithAssertTestingT(t assert.TestingT) {
	var err error
	assert.NilError(t, err)
}

func BenchmarkSomething(b *testing.B) {
	var err error
	assert.NilError(b, err)
}
`
	migration := newMigrationFromSource(t, source)
	migrateFile(migration)

	expected := `package foo

import (
	"testing"

	"github.com/go-check/check"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestWithChecker(c *check.C) {
	var err error
	assert.Check(c, cmp.NilError(err))
}

func HelperWithAssertTestingT(t assert.TestingT) {
	var err error
	assert.Check(t, cmp.NilError(err))
}

func BenchmarkSomething(b *testing.B) {
	var err error
	assert.Check(b, cmp.NilError(err))
}
`
	actual, err := formatFile(migration)
	assert.NilError(t, err)
	assert.Assert(t, cmp.EqualMultiLine(expected, string(actual)))
}
