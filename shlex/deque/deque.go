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
	return d.insert(&Element{Value: v}, at)
}

// Back returns the last element of list l or nil if the list is empty.
func (d *Deque) Back() *Element {
	if d.len == 0 {
		return nil
	}
	return d.root.prev
}

func (d *Deque) remove(e *Element) *Element {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	d.len--
	return e
}

func (d *Deque) Remove(e *Element) rune {
	d.remove(e)
	return e.Value
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
