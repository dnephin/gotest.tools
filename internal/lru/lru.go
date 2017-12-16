package lru

import (
	"bytes"
	"fmt"
	"go/ast"
	"sort"
	"strings"
)

type key string

type record struct {
	key    key
	value  ast.Node
	weight uint
}

func (r *record) String() string {
	return fmt.Sprintf("%s(%d)", r.key, r.weight)
}

type records struct {
	weight uint
	items  []*record
}

func (r *records) nextWeight() uint {
	ret := r.weight
	r.weight++
	return ret
}

func (r *records) Update(rec *record) {
	index := r.index(rec)
	r.items[index].weight = r.nextWeight()
	sort.Sort(r)
}

func (r *records) Push(rec *record) {
	rec.weight = r.nextWeight()
	r.items = append(r.items, rec)
}

func (r *records) Pop() *record {
	if len(r.items) == 0 {
		return nil
	}

	var first *record
	first, r.items = r.items[0], r.items[1:]
	return first
}

func (r *records) Len() int {
	return len(r.items)
}

func (r *records) Less(i, j int) bool {
	return r.items[i].weight < r.items[j].weight
}

func (r *records) Swap(i, j int) {
	r.items[i], r.items[j] = r.items[j], r.items[i]
}

func (r *records) index(rec *record) int {
	return sort.Search(r.Len(), func(i int) bool {
		return r.items[i].weight >= rec.weight
	})
}

func (r *records) String() string {
	buf := bytes.NewBufferString("records[")
	for _, rec := range r.items {
		buf.WriteString(rec.String() + " ")
	}
	return strings.TrimSpace(buf.String()) + "]"
}

type lru struct {
	cache map[key]*record
	used  *records
	cap   int
}

// Get an item from the cache
func (l *lru) Get(k key) (ast.Node, bool) {
	record, ok := l.cache[k]
	if ok {
		return l.setUsed(record).value, true
	}
	return nil, false
}

// Set an item in the cache
func (l *lru) Set(k key, value ast.Node) {
	l.cache[k] = l.setUsed(&record{key: k, value: value})
}

func (l *lru) setUsed(rec *record) *record {
	if rec, exists := l.cache[rec.key]; exists {
		l.used.Update(rec)
		return rec
	}

	if len(l.cache) == l.cap {
		remove := l.used.Pop()
		delete(l.cache, remove.key)
	}

	l.used.Push(rec)
	return rec
}

// NewLRU returns a new LRU cache
func NewLRU(cap int) *lru {
	lru := &lru{
		cap:   cap,
		cache: make(map[key]*record),
		used:  new(records),
	}
	return lru
}
