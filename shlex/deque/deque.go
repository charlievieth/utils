package deque

type Element struct {
	// Next and previous pointers in the doubly-linked list of elements.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.root is both the next element of the last
	// list element (l.Back()) and the previous element of the first list
	// element (l.Front()).
	next, prev *Element

	// The value stored with this element.
	Value rune
}

type Deque struct {
	root Element
	slab slab
	len  int
}

func (d *Deque) Init() *Deque {
	d.root.next = &d.root
	d.root.prev = &d.root
	d.len = 0
	return d
}

func (d *Deque) Len() int { return d.len }

func (d *Deque) Front() *Element {
	if d.len == 0 {
		return nil
	}
	return d.root.next
}

func (d *Deque) lazyInit() {
	if d.root.next == nil {
		d.Init()
	}
}

func (d *Deque) insert(e, at *Element) *Element {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	d.len++
	return e
}

func (d *Deque) insertValue(v rune, at *Element) *Element {
	e := d.slab.next()
	e.Value = v
	return d.insert(e, at)
}

// Back returns the last element of list l or nil if the list is empty.
func (d *Deque) Back() *Element {
	if d.len == 0 {
		return nil
	}
	return d.root.prev
}

func (d *Deque) remove(e *Element) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	d.len--
	d.slab.remove(e)
}

func (d *Deque) Remove(e *Element) rune {
	v := e.Value
	d.remove(e)
	return v
}

func (d *Deque) PushFront(v rune) *Element {
	d.lazyInit()
	return d.insertValue(v, &d.root)
}

func (d *Deque) PushBack(v rune) *Element {
	d.lazyInit()
	return d.insertValue(v, d.root.prev)
}

func (d *Deque) Pop() rune { return d.Remove(d.Back()) }

func (d *Deque) PopLeft() rune { return d.Remove(d.Front()) }

const slabSize = 8

// TODO: do we get any speedup from using a custom allocator???
type slab struct {
	len   int
	cap   int
	elems []*[slabSize]Element
}

func (s *slab) grow() {
	s.elems = append(s.elems, new([slabSize]Element))
	s.cap += slabSize
}

func (s *slab) next() *Element {
	if s.len == s.cap {
		s.grow()
	}
	var zero Element
	for _, p := range s.elems {
		for i := range p {
			if p[i] == zero {
				s.len++
				return &p[i]
			}
		}
	}
	return nil // trigger panic
}

func (s *slab) remove(e *Element) {
	*e = Element{}
	s.len--
}
