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

//go:linkname check intsets.check
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

func (s *LockingSparseSet) Copy(x *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.Copy(x)
	s.mu.Unlock()
}

func (s *LockingSparseSet) IntersectionWith(x *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.IntersectionWith(x)
	s.mu.Unlock()
}

func (s *LockingSparseSet) Intersection(x, y *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.Intersection(x, y)
	s.mu.Unlock()
}

func (s *LockingSparseSet) Intersects(x *intsets.Sparse) bool {
	s.mu.RLock()
	ok := s.sparse.Intersects(x)
	s.mu.RUnlock()
	return ok
}

func (s *LockingSparseSet) UnionWith(x *intsets.Sparse) bool {
	s.mu.Lock()
	ok := s.sparse.UnionWith(x)
	s.mu.Unlock()
	return ok
}

func (s *LockingSparseSet) Union(x, y *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.Union(x, y)
	s.mu.Unlock()
}

func (s *LockingSparseSet) DifferenceWith(x *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.DifferenceWith(x)
	s.mu.Unlock()
}

func (s *LockingSparseSet) Difference(x, y *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.Difference(x, y)
	s.mu.Unlock()
}

func (s *LockingSparseSet) SymmetricDifferenceWith(x *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.SymmetricDifferenceWith(x)
	s.mu.Unlock()
}

func (s *LockingSparseSet) SymmetricDifference(x, y *intsets.Sparse) {
	s.mu.Lock()
	s.sparse.SymmetricDifference(x, y)
	s.mu.Unlock()
}

func (s *LockingSparseSet) SubsetOf(x *intsets.Sparse) bool {
	s.mu.RLock()
	v := s.sparse.SubsetOf(x)
	s.mu.RUnlock()
	return v
}

func (s *LockingSparseSet) Equals(t *intsets.Sparse) bool {
	s.mu.RLock()
	v := s.sparse.Equals(t)
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
