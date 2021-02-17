package fastwalk

import (
	"sync"
	_ "unsafe"

	"golang.org/x/tools/container/intsets"
)

type LockingSparseSet struct {
	mu     sync.RWMutex
	sparse intsets.Sparse
}

// func NewLockingSparseSet() *LockingSparseSet {
// 	return nil
// }

func (s *LockingSparseSet) IsEmpty() bool {
	s.mu.RLock()
	v := s.sparse.IsEmpty()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) Len() int {
	s.mu.RLock()
	v := s.sparse.Len()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) Max() int {
	s.mu.RLock()
	v := s.sparse.Max()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) Min() int {
	s.mu.RLock()
	v := s.sparse.Min()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) LowerBound(x int) int {
	s.mu.RLock()
	v := s.sparse.LowerBound(x)
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) Insert(x int) bool {
	s.mu.Lock()
	ok := s.sparse.Insert(x)
	s.mu.Unlock()
	return ok
}

func (s *LockingSparseSet) Remove(x int) bool {
	s.mu.Lock()
	ok := s.sparse.Remove(x)
	s.mu.Unlock()
	return ok
}

func (s *LockingSparseSet) Clear() {
	s.mu.Lock()
	s.sparse.Clear()
	s.mu.Unlock()
}

func (s *LockingSparseSet) TakeMin(p *int) bool {
	s.mu.Lock()
	ok := s.sparse.TakeMin(p)
	s.mu.Unlock()
	return ok
}

//go:linkname check golang.org/x/tools/container/intsets.(*Sparse).check
func check(s *intsets.Sparse) error

func (s *LockingSparseSet) Check() error {
	return check(&s.sparse)
}

// func (s *LockingSparseSet) Check() error

func (s *LockingSparseSet) Has(x int) bool {
	s.mu.RLock()
	ok := s.sparse.Has(x)
	s.mu.RUnlock()
	return ok
}

func (s *LockingSparseSet) Copy(x *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	s.sparse.Copy(&x.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) IntersectionWith(x *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	s.sparse.IntersectionWith(&x.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) Intersection(x, y *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	if s != y {
		y.mu.RLock()
	}
	s.sparse.Intersection(&x.sparse, &y.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	if s != y {
		y.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) Intersects(x *LockingSparseSet) bool {
	s.mu.RLock()
	if s != x {
		x.mu.RLock()
	}
	ok := s.sparse.Intersects(&x.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	s.mu.RUnlock()
	return ok
}

func (s *LockingSparseSet) UnionWith(x *LockingSparseSet) bool {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	ok := s.sparse.UnionWith(&x.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	s.mu.Unlock()
	return ok
}

func (s *LockingSparseSet) Union(x, y *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	if s != y {
		y.mu.RLock()
	}
	s.sparse.Union(&x.sparse, &y.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	if s != y {
		y.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) DifferenceWith(x *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	s.sparse.DifferenceWith(&x.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) Difference(x, y *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	if s != y {
		y.mu.RLock()
	}
	s.sparse.Difference(&x.sparse, &y.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	if s != y {
		y.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) SymmetricDifferenceWith(x *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	s.sparse.SymmetricDifferenceWith(&x.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) SymmetricDifference(x, y *LockingSparseSet) {
	s.mu.Lock()
	if s != x {
		x.mu.RLock()
	}
	if s != y {
		y.mu.RLock()
	}
	s.sparse.SymmetricDifference(&x.sparse, &y.sparse)
	if s != x {
		x.mu.RUnlock()
	}
	if s != y {
		y.mu.RUnlock()
	}
	s.mu.Unlock()
}

func (s *LockingSparseSet) SubsetOf(x *LockingSparseSet) bool {
	s.mu.RLock()
	x.mu.RLock()
	v := s.sparse.SubsetOf(&x.sparse)
	x.mu.RUnlock()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) Equals(t *LockingSparseSet) bool {
	s.mu.RLock()
	t.mu.RLock()
	v := s.sparse.Equals(&t.sparse)
	t.mu.RUnlock()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) String() string {
	s.mu.RLock()
	v := s.sparse.String()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) BitString() string {
	s.mu.RLock()
	v := s.sparse.BitString()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) GoString() string {
	s.mu.RLock()
	v := s.sparse.GoString()
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) AppendTo(slice []int) []int {
	s.mu.Lock()
	v := s.sparse.AppendTo(slice)
	s.mu.Unlock()
	return v
}
