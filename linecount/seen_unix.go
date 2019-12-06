// +build linux,!appengine darwin freebsd openbsd netbsd

package main

import (
	"os"
	"sync"
	"syscall"
)

type fileKey struct {
	Dev uint64
	Ino uint64
}

type SeenFiles struct {
	keys map[fileKey]struct{}
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu sync.Mutex
}

func (s *SeenFiles) Seen(path string) bool {
	// TODO (CEV): we can use syscall.Stat() directly
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	stat := fi.Sys().(*syscall.Stat_t)
	key := fileKey{
		Dev: uint64(stat.Dev),
		Ino: stat.Ino,
	}
	s.mu.Lock()
	if s.keys == nil {
		s.keys = make(map[fileKey]struct{})
	}
	_, ok := s.keys[key]
	if !ok {
		s.keys[key] = struct{}{}
	}
	s.mu.Unlock()
	return ok
}
