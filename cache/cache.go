package main

import (
	"hash/maphash"
	"sync"
	"sync/atomic"
	"unsafe"
)

// An entry is a slot in the map corresponding to a particular key.
type entry struct {
	// p points to the interface{} value stored for the entry.
	//
	// If p == nil, the entry has been deleted and m.dirty == nil.
	//
	// If p == expunged, the entry has been deleted, m.dirty != nil, and the entry
	// is missing from m.dirty.
	//
	// Otherwise, the entry is valid and recorded in m.read.m[key] and, if m.dirty
	// != nil, in m.dirty[key].
	//
	// An entry can be deleted by atomic replacement with nil: when m.dirty is
	// next created, it will atomically replace nil with expunged and leave
	// m.dirty[key] unset.
	//
	// An entry's associated value can be updated by atomic replacement, provided
	// p != expunged. If p == expunged, an entry's associated value can be updated
	// only after first setting m.dirty[key] = e so that lookups using the dirty
	// map find the entry.
	p unsafe.Pointer // *interface{}
}

func newEntry(i interface{}) *entry {
	return &entry{p: unsafe.Pointer(&i)}
}

func (e *entry) load() (value interface{}, ok bool) {
	// TODO: we can probably just make this a Load()
	if p := atomic.LoadPointer(&e.p); p != nil {
		return *(*interface{})(p), true
	}
	return nil, false
}

func (e *entry) store(i *interface{}) {
	atomic.StorePointer(&e.p, unsafe.Pointer(i))
}

func (e *entry) delete() {
	atomic.StorePointer(&e.p, nil)
}

// TODO: do we really need this?
func (e *entry) tryStore(i *interface{}) {
	for {
		p := atomic.LoadPointer(&e.p)
		if atomic.CompareAndSwapPointer(&e.p, p, unsafe.Pointer(i)) {
			return
		}
	}
}

type atomicShard struct {
	mu sync.RWMutex
	m  map[string]*entry
}

func (s *atomicShard) Delete(key string) {
	s.mu.Lock()
	if e, ok := s.m[key]; ok {
		e.delete()
		delete(s.m, key)
	}
	s.mu.Unlock()
}

func (s *atomicShard) Load(key string) (value interface{}, ok bool) {
	s.mu.RLock()
	e, ok := s.m[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return e.load()
}

func (s *atomicShard) LoadOrStore(key string, value interface{}) (actual interface{}, loaded bool) {
	s.mu.RLock()
	e, loaded := s.m[key]
	s.mu.RUnlock()
	if loaded {
		actual, _ = e.load()
		return actual, true
	}

	s.mu.Lock()
	e, loaded = s.m[key]
	if loaded {
		actual, _ = e.load()
	} else {
		s.m[key] = newEntry(value)
		actual, loaded = value, false
	}
	s.mu.Unlock()
	return actual, loaded
}

func (s *atomicShard) Store(key string, value interface{}) {
	s.mu.RLock()
	e, ok := s.m[key]
	s.mu.RUnlock()
	if ok {
		e.tryStore(&value)
		return
	}

	s.mu.Lock()
	if e, ok := s.m[key]; ok {
		e.tryStore(&value)
	} else {
		s.m[key] = newEntry(value)
	}
	s.mu.Unlock()
	return
}

type shard struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

func (s *shard) Delete(key string) {
	s.mu.Lock()
	delete(s.m, key)
	s.mu.Unlock()
}

func (s *shard) Load(key string) (value interface{}, ok bool) {
	s.mu.RLock()
	value, ok = s.m[key]
	s.mu.RUnlock()
	return
}

func (s *shard) Store(key string, val interface{}) {
	s.mu.Lock()
	s.m[key] = val
	s.mu.Unlock()
}

func (s *shard) LoadOrStore(key string, val interface{}) (actual interface{}, loaded bool) {
	s.mu.RLock()
	v, ok := s.m[key]
	s.mu.RUnlock()
	if ok {
		return v, false
	}

	s.mu.Lock()
	v, ok = s.m[key]
	if !ok {
		s.m[key] = val
		v = val
	}
	s.mu.Unlock()
	return v, ok
}

type Cache struct {
	seed   maphash.Seed
	shards [256]shard
}

func (c *Cache) shard(key string) *shard {
	var h maphash.Hash
	h.SetSeed(c.seed)
	h.WriteString(key)
	return &c.shards[h.Sum64()%uint64(len(c.shards))]
}

func (c *Cache) Store(key string, val interface{}) {
	c.shard(key).Store(key, val)
}

func (c *Cache) LoadOrStore(key string, val interface{}) {
	c.shard(key).LoadOrStore(key, val)
}

type Node struct {
	Value interface{}
	next  *Node
}

type List struct {
	head *Node
	tail *Node
}

func NewList() *List {
	return &List{head: new(Node), tail: new(Node)}
}

func (l *List) WalkFast() int {
	n := l.head
	i := 0
	for n != nil {
		n = n.next
		i++
	}
	return i
}

func (l *List) Walk() int {
	p := unsafe.Pointer(l.head)
	n := (*Node)(atomic.LoadPointer(&p))
	i := 0
	for n != nil {
		p = unsafe.Pointer(n.next)
		n = (*Node)(atomic.LoadPointer(&p))
		i++
	}
	return i
}

// NOTE: when walking atomically make sure we don't get stuck in an infinite loop

func (l *List) move(e, at *Node) {
	if e == at {
		return
	}
	e.next = l.head.next
	l.head.next = e
}

func (l *List) PushFront(v interface{}) {
	e := &Node{Value: v}
	e.next = l.head.next
	l.head.next = e
}

func (l *List) MoveToFront(n *Node) {
	//
}

func (l *List) Insert(value interface{}) {

}

func main() {
	ll := NewList()
	for i := 0; i < 10; i++ {
		ll.PushFront(i)
	}

}
