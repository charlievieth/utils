// +build appengine !linux,!darwin,!freebsd,!openbsd,!netbsd

package main

import (
	"path/filepath"
	"sync"
)

type SeenFiles struct {
	seen map[string]struct{}
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu sync.Mutex
}

func (s *SeenFiles) Seen(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	name, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	var ok bool
	s.mu.Lock()
	if s.seen != nil {
		_, ok = s.seen[name]
	} else {
		s.seen = make(map[string]struct{})
	}
	if !ok {
		s.seen[name] = struct{}{}
	}
	s.mu.Unlock()
	return ok
}
