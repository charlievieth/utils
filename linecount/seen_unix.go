// +build linux,!appengine darwin freebsd openbsd netbsd

package main

import (
	"sync"
	"syscall"
)

type fileKey struct {
	Dev uint64
	Ino uint64
}

type SeenFiles struct {
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu   sync.Mutex
	keys map[fileKey]struct{}
}

func (s *SeenFiles) Path(path string) bool {
	// TODO (CEV): we can use syscall.Stat() directly
	var stat syscall.Stat_t
	if syscall.Stat(path, &stat) != nil {
		return false
	}
	key := fileKey{
		Dev: uint64(stat.Dev),
		Ino: uint64(stat.Ino),
	}
	s.mu.Lock()
	if s.keys == nil {
		s.keys = make(map[fileKey]struct{}, 1024)
	}
	_, ok := s.keys[key]
	if !ok {
		s.keys[key] = struct{}{}
	}
	s.mu.Unlock()
	return ok
}
