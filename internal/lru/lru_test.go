package lru

import (
	"go/token"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/assert/cmp"
)

type node struct {
	start token.Pos
	end   token.Pos
}

func (n *node) Pos() token.Pos {
	return n.start
}
func (n *node) End() token.Pos {
	return n.end
}

func TestLRUCacheGetAndSet(t *testing.T) {
	assert := assert.New(t)
	c := newLRU(3)

	// missing
	item, ok := c.get("a")
	assert.Assert(!ok)
	assert.Equal(item, nil)

	item = &node{1, 2}
	c.set("b", item)
	stored, ok := c.get("b")
	assert.Assert(ok)
	assert.Equal(item, stored)
}

func TestLRUCacheSetAtCap(t *testing.T) {
	assert := assert.New(t)
	c := newLRU(3)
	x := &node{3, 4}

	c.set("a", x)
	c.set("b", x)
	c.set("b", x)
	c.set("b", x)
	c.set("c", x)

	assert.Assert(cmp.Compare(usedKeys(c.used), []string{"c", "b", "a"}))

	_, ok := c.get("a")
	assert.Assert(ok)

	c.set("d", x)
	assert.Assert(cmp.Compare(usedKeys(c.used), []string{"d", "a", "c"}))
}

func usedKeys(u *records) []string {
	keys := []string{}
	for _, rec := range u.items {
		keys = append(keys, string(rec.key))
	}
	return keys
}
