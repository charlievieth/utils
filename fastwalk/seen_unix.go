// +build linux,!appengine darwin freebsd openbsd netbsd !windows

package fastwalk

import (
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/tools/container/intsets"
)

type seenFiles struct {
	mu   sync.RWMutex
	sets map[uint64]*sparseIntset
}

type sparseIntset struct {
	mu     sync.Mutex
	sparse intsets.Sparse
}

func (s *sparseIntset) Seen(ino uint64) bool {
	s.mu.Lock()
	seen := !s.sparse.Insert(int(ino))
	s.mu.Unlock()
	return seen
}

func (s *seenFiles) Set(dev uint64) *sparseIntset {
	s.mu.RLock()
	set, ok := s.sets[dev]
	s.mu.RUnlock()
	if ok {
		return set
	}

	s.mu.Lock()
	if s.sets == nil {
		s.sets = make(map[uint64]*sparseIntset)
	}
	if set, ok = s.sets[dev]; !ok {
		set = &sparseIntset{}
		s.sets[dev] = set
	}
	s.mu.Unlock()
	return set
}

func (s *seenFiles) SeenAt(fd int, name string) bool {
	var stat unix.Stat_t
	// set: AT_SYMLINK_NOFOLLOW for Lstat
	if unix.Fstatat(fd, name, &stat, 0) != nil {
		return false
	}
	return s.Set(uint64(stat.Dev)).Seen(uint64(stat.Ino))
}

func (s *seenFiles) Seen(path string) bool {
	var stat syscall.Stat_t
	if syscall.Stat(path, &stat) != nil {
		return false
	}
	return s.Set(uint64(stat.Dev)).Seen(uint64(stat.Ino))
}

/*
type fileKey struct {
	Dev uint64
	Ino uint64
}

// TODO: use multiple maps based with dev/ino hash
// to reduce lock contention.
type seenFiles struct {
	keys map[fileKey]struct{}
	// we assume most files have not been seen so
	// no need for a RWMutex
	mu sync.Mutex
}

func newSeenFiles() *seenFiles {
	return &seenFiles{keys: make(map[fileKey]struct{}, 512)}
}

// WARN: rename
// func (s *seenFiles) Ent(dev, ino uint64) bool {
// 	return false
// }

func (s *seenFiles) SeenAt(fd int, name string) bool {
	var stat unix.Stat_t
	// set: AT_SYMLINK_NOFOLLOW for Lstat
	if unix.Fstatat(fd, name, &stat, 0) != nil {
		return false
	}
	return s.seen(uint64(stat.Dev), uint64(stat.Ino))
}

func (s *seenFiles) seen(dev, ino uint64) bool {
	key := fileKey{
		Dev: dev,
		Ino: ino,
	}
	s.mu.Lock()
	// if s.keys == nil {
	// 	s.keys = make(map[fileKey]struct{}, 1024)
	// }
	_, ok := s.keys[key]
	if !ok {
		s.keys[key] = struct{}{}
	}
	s.mu.Unlock()
	return ok
}

func (s *seenFiles) Path(path string) bool {
	// TODO (CEV): we can use syscall.Stat() directly
	var stat syscall.Stat_t
	if syscall.Stat(path, &stat) != nil {
		return false
	}
	// fi, err := os.Stat(path)
	// if err != nil {
	// 	return false
	// }
	// stat := fi.Sys().(*syscall.Stat_t)
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
*/

// func IsSymLink(path string) bool {
// 	var stat syscall.Stat_t
// 	if syscall.Stat(path, &stat) != nil {
// 		return false
// 	}
// 	return stat.Mode&syscall.S_IFMT == syscall.S_IFLNK
// }

// TODO: consider using something like this
// type Walker_XXX struct {
// 	fileFn         func()
// 	dirFn          func()
// 	followSymlinks bool
// }

/*
type seenFiles struct {
	buckets [8]*deviceMap
}

type deviceMap struct {
	keys map[fileKey]struct{}
	mu   sync.Mutex
}

func (m *deviceMap) seen(dev, ino uint64) bool {
	key := fileKey{
		Dev: dev,
		Ino: ino,
	}
	m.mu.Lock()
	if m.keys == nil {
		m.keys = make(map[fileKey]struct{}, 1024)
	}
	_, ok := m.keys[key]
	if !ok {
		m.keys[key] = struct{}{}
	}
	m.mu.Unlock()
	return ok
}
*/
