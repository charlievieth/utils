package main

import (
	"fmt"
)

// A Sparse is a set of int values.
// Sparse operations (even queries) are not concurrency-safe.
//
// The zero value for Sparse is a valid empty set.
//
// Sparse sets must be copied using the Copy method, not by assigning
// a Sparse value.
//
type Sparse struct {
	// An uninitialized Sparse represents an empty set.
	// An empty set may also be represented by
	//  root.next == root.prev == &root.
	// In a non-empty set, root.next points to the first block and
	// root.prev to the last.
	// root.offset and root.bits are unused.
	root block
}

type word uintptr

const (
	_m            = ^word(0)
	bitsPerWord   = 8 << (_m>>8&1 + _m>>16&1 + _m>>32&1)
	bitsPerBlock  = 256 // optimal value for go/pointer solver performance
	wordsPerBlock = bitsPerBlock / bitsPerWord
)

// Limit values of implementation-specific int type.
const (
	MaxInt = int(^uint(0) >> 1)
	MinInt = -MaxInt - 1
)

// -- block ------------------------------------------------------------

// A set is represented as a circular doubly-linked list of blocks,
// each containing an offset and a bit array of fixed size
// bitsPerBlock; the blocks are ordered by increasing offset.
//
// The set contains an element x iff the block whose offset is x - (x
// mod bitsPerBlock) has the bit (x mod bitsPerBlock) set, where mod
// is the Euclidean remainder.
//
// A block may only be empty transiently.
//
type block struct {
	offset     int                 // offset mod bitsPerBlock == 0
	bits       [wordsPerBlock]word // contains at least one set bit
	next, prev *block              // doubly-linked list of blocks
}

// wordMask returns the word index (in block.bits)
// and single-bit mask for the block's ith bit.
func wordMask(i uint) (w uint, mask word) {
	w = i / bitsPerWord
	mask = 1 << (i % bitsPerWord)
	return
}

// insert sets the block b's ith bit and
// returns true if it was not already set.
//
func (b *block) insert(i uint) bool {
	w, mask := wordMask(i)
	if b.bits[w]&mask == 0 {
		b.bits[w] |= mask
		return true
	}
	return false
}

// offsetAndBitIndex returns the offset of the block that would
// contain x and the bit index of x within that block.
//
func offsetAndBitIndex(x int) (int, uint) {
	mod := x % bitsPerBlock
	if mod < 0 {
		// Euclidean (non-negative) remainder
		mod += bitsPerBlock
	}
	return x - mod, uint(mod)
}

// -- Sparse --------------------------------------------------------------

// start returns the root's next block, which is the root block
// (if s.IsEmpty()) or the first true block otherwise.
// start has the side effect of ensuring that s is properly
// initialized.
//
func (s *Sparse) start() *block {
	root := &s.root
	if root.next == nil {
		root.next = root
		root.prev = root
	} else if root.next.prev != root {
		// Copying a Sparse x leads to pernicious corruption: the
		// new Sparse y shares the old linked list, but iteration
		// on y will never encounter &y.root so it goes into a
		// loop.  Fail fast before this occurs.
		panic("A Sparse has been copied without (*Sparse).Copy()")
	}

	return root.next
}

// Insert adds x to the set s, and reports whether the set grew.
func (s *Sparse) Insert(x int) bool {
	offset, i := offsetAndBitIndex(x)
	b := s.start()
	for b != &s.root && b.offset <= offset {
		if b.offset == offset {
			return b.insert(i)
		}
		b = b.next
	}

	// Insert new block before b.
	new := &block{offset: offset}
	new.next = b
	new.prev = b.prev
	new.prev.next = new
	new.next.prev = new
	return new.insert(i)
}

// block returns the block that would contain offset,
// or nil if s contains no such block.
//
func (s *Sparse) block(offset int) *block {
	b := s.start()
	for b != &s.root && b.offset <= offset {
		if b.offset == offset {
			return b
		}
		b = b.next
	}
	return nil
}

// has reports whether the block's ith bit is set.
func (b *block) has(i uint) bool {
	w, mask := wordMask(i)
	return b.bits[w]&mask != 0
}

// Has reports whether x is an element of the set s.
func (s *Sparse) Has(x int) bool {
	offset, i := offsetAndBitIndex(x)
	if b := s.block(offset); b != nil {
		return b.has(i)
	}
	return false
}

func main() {
	var s Sparse
	s.Insert(1)
	s.Insert(2)
	s.Insert(3)
	fmt.Println(s.Has(1))
	fmt.Println(s.Has(2))
	fmt.Println(s.Has(3))
	fmt.Println(s.Has(4))
}
