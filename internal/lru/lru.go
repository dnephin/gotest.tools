package lru

import (
	"bytes"
	"container/heap"
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

func (r *records) increaseWeight(i int) {
	r.items[i].weight = r.nextWeight()
}

func (r *records) Push(x interface{}) {
	record := x.(*record)
	record.weight = r.nextWeight()
	r.items = append(r.items, record)
}

func (r *records) Pop() interface{} {
	index := len(r.items) - 1
	if index < 0 {
		return nil
	}

	var last *record
	last, r.items = r.items[index], r.items[:index]
	return last
}

func (r *records) Len() int {
	return len(r.items)
}

func (r *records) Less(i, j int) bool {
	return r.items[i].weight > r.items[j].weight
}

func (r *records) Swap(i, j int) {
	r.items[i], r.items[j] = r.items[j], r.items[i]
}

func (r *records) index(rec *record) int {
	return sort.Search(r.Len(), func(i int) bool {
		return r.items[i].weight <= rec.weight
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

func (l *lru) get(k key) (ast.Node, bool) {
	record, ok := l.cache[k]
	if ok {
		return l.setUsed(record).value, true
	}
	return nil, false
}

func (l *lru) set(k key, value ast.Node) {
	l.cache[k] = l.setUsed(&record{key: k, value: value})
}

func (l *lru) setUsed(rec *record) *record {
	if rec, exists := l.cache[rec.key]; exists {
		index := l.used.index(rec)
		l.used.increaseWeight(index)
		heap.Fix(l.used, index)
		fmt.Printf("FIX %s\n", l.used)
		return rec
	}

	if len(l.cache) == l.cap {
		remove := heap.Remove(l.used, l.used.Len()-1).(*record)
		delete(l.cache, remove.key)
		fmt.Printf("REMOVE %s\n", l.used)
	}

	heap.Push(l.used, rec)
	// FIXME: Why is this sort necessary?
	sort.Sort(l.used)
	fmt.Printf("PUSH %s\n", l.used)
	return rec
}

func newLRU(cap int) *lru {
	lru := &lru{
		cap:   cap,
		cache: make(map[key]*record),
		used:  new(records),
	}
	heap.Init(lru.used)
	return lru
}
