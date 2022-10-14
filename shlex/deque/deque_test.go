package deque

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushFront(t *testing.T) {
	var d Deque
	d.PushFront(1)
	d.PushFront(2)
	d.PushFront(3)
	assert.Equal(t, 3, d.Len())
	assert.Equal(t, rune(3), d.PopLeft())
	assert.Equal(t, rune(1), d.Pop())
	assert.Equal(t, rune(2), d.PopLeft())
	assert.Equal(t, 0, d.Len())
	assert.Nil(t, d.Front())
	assert.Nil(t, d.Back())
}

func TestPushBack(t *testing.T) {
	var d Deque
	d.PushBack(1)
	d.PushBack(2)
	d.PushBack(3)
	assert.Equal(t, 3, d.Len())
	assert.Equal(t, rune(3), d.Pop())
	assert.Equal(t, rune(1), d.PopLeft())
	assert.Equal(t, rune(2), d.Pop())
	assert.Equal(t, 0, d.Len())
	assert.Nil(t, d.Front())
	assert.Nil(t, d.Back())
}

func TestDequeSlab(t *testing.T) {
	var d Deque
	runes := make([]rune, 512)
	for i := 0; i < len(runes); i++ {
		runes[i] = rune(i)
		d.PushBack(rune(i))
	}
	for i := range runes {
		assert.Equal(t, runes[i], d.PopLeft())
	}

	for i := 0; i < len(runes); i++ {
		runes[i] = rune(i)
		d.PushFront(rune(i))
	}
	for i := range runes {
		assert.Equal(t, runes[i], d.Pop())
	}
}

func TestSlabGrow(t *testing.T) {
	var s slab
	var empty Element
	for i := 0; i < 9; i++ {
		e := s.next()
		require.NotNilf(t, e, "#%d", i)
		assert.Equalf(t, empty, *e, "#%d", i)
		// populate e so that it is no longer considered empty
		*e = Element{
			next:  &empty,
			prev:  &empty,
			Value: 'a',
		}
	}
	assert.Len(t, s.elems, 2)
	assert.Equal(t, 9, s.len)

	for i := 0; i < len(s.elems); i++ {
		for j := 0; j < len(s.elems[i]); j++ {
			if s.elems[i][j] != empty {
				s.remove(&s.elems[i][j])
			}
		}
	}
	assert.Equal(t, 0, s.len)
	for i := range s.elems {
		for j := range s.elems[i] {
			assert.Equalf(t, empty, s.elems[i][j], "#%d", i)
		}
	}
}

func BenchmarkDeque(b *testing.B) {
	var d Deque
	var cases [4]int
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			cases[0]++
			d.PushBack(1)
			d.PushBack(2)
			d.PushBack(3)
			d.PushBack(4)
			if d.Len() > 1000 {
				b.Fatal("Len:", d.Len())
			}
		case 1:
			cases[1]++
			d.PushFront(5)
			d.PushFront(6)
			d.PushFront(7)
			d.PushFront(8)
		case 2:
			cases[2]++
			d.Pop()
			d.PopLeft()
			d.Pop()
			d.PopLeft()
		case 3:
			cases[3]++
			d.Pop()
			d.PopLeft()
			d.Pop()
			d.PopLeft()
		}
	}
}

// Benchmark against container/list for reference
func BenchmarkList(b *testing.B) {
	var d list.List
	var cases [4]int
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			cases[0]++
			d.PushBack(1)
			d.PushBack(2)
			d.PushBack(3)
			d.PushBack(4)
			if d.Len() > 1000 {
				b.Fatal("Len:", d.Len())
			}
		case 1:
			cases[1]++
			d.PushFront(5)
			d.PushFront(6)
			d.PushFront(7)
			d.PushFront(8)
		case 2:
			cases[2]++
			d.Remove(d.Back())
			d.Remove(d.Front())
			d.Remove(d.Back())
			d.Remove(d.Front())
		case 3:
			cases[3]++
			d.Remove(d.Back())
			d.Remove(d.Front())
			d.Remove(d.Back())
			d.Remove(d.Front())
		}
	}
}
