package foo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mystruct struct {
	a        int
	expected int
}

func TestFirstThing(t *testing.T) {
	rt := require.TestingT(t)
	assert.Equal(t, "foo", "bar")
	assert.Equal(t, 1, 2)
	assert.True(t, false)
	assert.False(t, true)
	require.NoError(t, nil)

	assert.Equal(t, map[string]bool{"a": true}, nil)
	assert.Equal(t, []int{1}, nil)
}

func TestSecondThing(t *testing.T) {
	var foo mystruct
	require.Equal(t, foo, mystruct{})

	require.Equal(t, mystruct{}, mystruct{})

	assert.NoError(t, nil, "foo %d", 3)

	assert.Error(t, fmt.Errorf("foo"))
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
	assert.Equal(c, "A", "b")
}

func TestEqualsWithComplexTypes(t *testing.T) {
	expected := []int{1, 2, 3}
	assert.Equal(t, expected, nil)

	expectedM := map[int]bool{}
	assert.Equal(t, expectedM, nil)

	expectedI := 123
	assert.Equal(t, expectedI, 0)

	assert.Equal(t, doInt(), 3)
	// TODO: struct field
}

func doInt() int {
	return 1
}

func TestEqualWithPrimitiveTypes(t *testing.T) {
	s := "foo"
	ptrString := &s
	assert.Equal(t, *ptrString, "foo")

	assert.Equal(t, doInt(), doInt())

	x := doInt()
	y := doInt()
	assert.Equal(t, x, y)

	tc := mystruct{a: 3, expected: 5}
	assert.Equal(t, tc.a, tc.expected)
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
		assert.Equal(t, testcase.actual, testcase.expected)
		assert.Equal(t, testcase.opts, testcase.expectedOpts)
	}
}
