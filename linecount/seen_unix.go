// +build linux,!appengine darwin freebsd openbsd netbsd

package main

import (
	"os"
	"sync"
	"syscall"
)

type fileKey struct {
	Dev int32
	Ino uint64
}

type SeenFiles struct {
	keys map[fileKey]struct{}
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu sync.Mutex
}

func (s *SeenFiles) Seen(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	stat := fi.Sys().(*syscall.Stat_t)
	key := fileKey{
		Dev: stat.Dev,
		Ino: stat.Ino,
	}
	var ok bool
	s.mu.Lock()
	if s.keys != nil {
		_, ok = s.keys[key]
	} else {
		s.keys = make(map[fileKey]struct{})
	}
	if !ok {
		s.keys[key] = struct{}{}
	}
	s.mu.Unlock()
	return ok
}
