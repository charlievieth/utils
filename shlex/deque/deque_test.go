package deque

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
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
