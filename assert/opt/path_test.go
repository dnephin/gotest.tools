package opt

import (
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/gotestyourself/gotestyourself/assert"
)

type node struct {
	Value    int
	Labels   map[string]node
	Children []node
	Ref      *node
}

func TestSpecFromStruct(t *testing.T) {
	newNode := func(value int) node {
		return node{
			Ref: &node{
				Children: []node{
					{},
					{
						Labels: map[string]node{
							"first": {Value: value},
						},
					},
				},
			},
		}
	}

	path := Path(
		Type(node{}),
		Step(Field("Ref"), Type(&node{})),
		Indirect,
		Step(Field("Children"), Type([]node{})),
		Slice,
		Step(Field("Labels"), Type(map[string]node{})),
		MapKey("first"),
		Step(Field("Value"), Type(0)),
	)

	partial := PathPartial(
		Step(Field("Ref"), Type(&node{})),
		Step(Field("Children"), Type([]node{})),
		Step(Field("Labels"), Type(map[string]node{})),
		Step(Field("Value"), Type(0)),
	)

	opt := gocmp.FilterPath(path, gocmp.Ignore())
	assert.DeepEqual(t, newNode(1), newNode(2), opt)

	opt = gocmp.FilterPath(partial, gocmp.Ignore())
	assert.DeepEqual(t, newNode(1), newNode(2), opt)
}

func TestSpecFromSlice(t *testing.T) {
	newNodes := func(end node) []node {
		return []node{
			{
				Ref: &node{
					Children: []node{
						{},
						{
							Labels: map[string]node{
								"first": {},
								"second": {
									Ref: &end,
								},
							},
						},
					},
				},
			},
		}
	}

	path := Path(
		Type([]node{}),
		Slice,
		Step(Field("Ref"), Type(&node{})),
		Indirect,
		Step(Field("Children"), Type([]node{})),
		Slice,
		Step(Field("Labels"), Type(map[string]node{})),
		Map,
		Step(Field("Ref"), Type(&node{})),
		Indirect,
		Step(Field("Value"), Type(0)),
	)

	opt := gocmp.FilterPath(path, gocmp.Ignore())
	assert.DeepEqual(t, newNodes(node{Value: 2}), newNodes(node{Value: 3}), opt)
}
