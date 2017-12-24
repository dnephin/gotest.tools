package foo

import (
	"fmt"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

type mystruct struct {
}

func TestFirstThing(t *testing.T) {
	rt := assert.TestingT(t)
	assert.Check(t, cmp.Equal("foo", "bar"))
	assert.Check(t, cmp.Equal(1, 2))
	assert.Check(t, false)
	assert.Check(t, !true)
	assert.NilError(t, nil)
	assert.Check(t, cmp.Compare(map[string]bool{"a": true}, nil))
	assert.Check(t, cmp.Compare([]int{1}, nil))
}

func TestSecondThing(t *testing.T) {
	var foo mystruct
	assert.Assert(t, cmp.Compare(foo, mystruct{}))
	assert.Assert(t, cmp.Compare(mystruct{}, mystruct{}))
	assert.Check(t, cmp.NilError(nil), "foo %d", 3)
	assert.Check(t, cmp.ErrorContains(fmt.Errorf("foo"), ""))
}

func TestMissed(t *testing.T) {
	a := assert.New(t)

	a.Equal(t, "a", "b")
}

type unit struct {
	c *testing.T
}

func thing(t *testing.T) unit {
	return unit{c: t}
}

func TestStoredTestingT(t *testing.T) {
	u := thing(t)
	assert.Equal(u.c, "A", "b")
}

func TestNotNamedT(c *testing.T) {
	assert.Check(c, cmp.Equal("A", "b"))
}

func TestEqualsWithComplexTypes(t *testing.T) {
	expected := []int{1, 2, 3}
	assert.Check(t, cmp.Compare(expected, nil))

	expectedM := map[int]bool{}
	assert.Check(t, cmp.Compare(expectedM, nil))

	expectedI := 123
	assert.Check(t, cmp.Equal(expectedI, 0))
	assert.Check(t, cmp.Compare(doInt(), 3))
	// TODO: struct field
}

func doInt() int {
	return 1
}
