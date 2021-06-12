// +build appengine !linux,!darwin,!freebsd,!openbsd,!netbsd,!windows

package fastwalk

import (
	"path/filepath"
	"sync"
)

type seenFiles struct {
	seen map[string]struct{}
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu sync.Mutex
}

func (s *seenFiles) Seen(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	name, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return false
	}
	s.mu.Lock()
	if s.seen == nil {
		s.seen = make(map[string]struct{}, 512)
	}
	_, ok := s.seen[name]
	if !ok {
		s.seen[name] = struct{}{}
	}
	s.mu.Unlock()
	return ok
}
