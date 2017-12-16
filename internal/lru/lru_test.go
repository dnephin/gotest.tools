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
	c := NewLRU(3)

	// missing
	item, ok := c.Get("a")
	assert.Assert(!ok)
	assert.Equal(item, nil)

	item = &node{1, 2}
	c.Set("b", item)
	stored, ok := c.Get("b")
	assert.Assert(ok)
	assert.Equal(item, stored)
}

func TestLRUCacheSetAtCap(t *testing.T) {
	assert := assert.New(t)
	c := NewLRU(3)
	x := &node{3, 4}

	c.Set("a", x)
	c.Set("b", x)
	c.Set("b", x)
	c.Set("b", x)
	c.Set("c", x)

	assert.Assert(cmp.Compare(usedKeys(c.used), []string{"a", "b", "c"}))

	_, ok := c.Get("a")
	assert.Assert(ok)

	c.Set("d", x)
	assert.Assert(cmp.Compare(usedKeys(c.used), []string{"c", "a", "d"}))
}

func usedKeys(u *records) []string {
	keys := []string{}
	for _, rec := range u.items {
		keys = append(keys, string(rec.key))
	}
	return keys
}
