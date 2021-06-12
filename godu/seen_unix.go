// +build linux,!appengine darwin freebsd openbsd netbsd

package main

import (
	"sync"
	"syscall"
)

type fileEntry struct {
	Dev uint64
	Ino uint64
}

type SymlinkWatcher struct {
	mu   sync.Mutex
	ents map[fileEntry]struct{}
}

func (s *SymlinkWatcher) Seen(stat *syscall.Stat_t) bool {
	s.mu.Lock()
	if s.ents == nil {
		s.ents = make(map[fileEntry]struct{}, 1024)
	}
	ent := fileEntry{Dev: uint64(stat.Dev), Ino: uint64(stat.Ino)}
	_, seen := s.ents[ent]
	if !seen {
		s.ents[ent] = struct{}{}
	}
	s.mu.Unlock()
	return seen
}

// Dev           int32
// Mode          uint16
// Nlink         uint16
// Ino           uint64
