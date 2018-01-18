package foo

import (
	"fmt"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

type mystruct struct {
	a        int
	expected int
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
	assert.Check(u.c, cmp.Equal("A", "b"))

	u = unit{c: t}
	assert.Check(u.c, cmp.Equal("A", "b"))
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

	assert.Check(t, cmp.Equal(doInt(), 3))
	// TODO: struct field
}

func doInt() int {
	return 1
}

func TestEqualWithPrimitiveTypes(t *testing.T) {
	s := "foo"
	ptrString := &s
	assert.Check(t, cmp.Equal(*ptrString, "foo"))

	assert.Check(t, cmp.Equal(doInt(), doInt()))

	x := doInt()
	y := doInt()
	assert.Check(t, cmp.Equal(x, y))

	tc := mystruct{a: 3, expected: 5}
	assert.Check(t, cmp.Equal(tc.a, tc.expected))
}

func TestTableTest(t *testing.T) {
	var testcases = []struct {
		opts         []string
		actual       string
		expected     string
		expectedOpts []string
	}{
		{
			opts:     []string{"a", "b"},
			actual:   "foo",
			expected: "else",
		},
	}

	for _, testcase := range testcases {
		assert.Check(t, cmp.Equal(testcase.actual, testcase.expected))
		assert.Check(t, cmp.Compare(testcase.opts, testcase.expectedOpts))
	}
}
