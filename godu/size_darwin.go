// +build darwin

package main

// #cgo CFLAGS: -O3 -march=native -mtune=native
//
// #include <stdlib.h>
// #include <stdint.h>
// #include <sys/stat.h>
//
// #define unlikely(x) __builtin_expect(!!(x), 0)
//
// static int64_t file_size(const char *path) {
// 	struct stat st;
// 	if (unlikely(lstat(path, &st) != 0)) {
// 		return 0;
// 	}
// 	return (int64_t)st.st_size;
// }
//
import "C"

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

type buffer struct {
	// make sure the buffer is exactly 4096 bytes in size
	buf [4096 - unsafe.Sizeof([]byte{})]byte
	tmp []byte
}

func (b *buffer) WritePath(path string) {
	if len(path) < len(b.buf) {
		copy(b.buf[:], path)
		b.buf[len(path)] = 0
	} else {
		b.tmp = make([]byte, len(path)+1)
		copy(b.tmp, path)
	}
}

func (b *buffer) Pointer() unsafe.Pointer {
	if b.tmp == nil {
		return unsafe.Pointer(&b.buf[0])
	}
	return unsafe.Pointer(&b.tmp[0])
}

func (b *buffer) Free() {
	b.tmp = nil
	bufPool.Put(b)
}

var bufPool = sync.Pool{
	New: func() interface{} { return &buffer{} },
}

// TODO: pass the file descriptor instead of the path
//
// Nasty hack to avoid using the libc_lstat64_trampoline, which is dog slow.
func GetFileSize(path string) (int64, error) {
	// TODO: use fstat() and the FD in fastwalk
	// TODO: we can also just call stat directly ???

	p := bufPool.Get().(*buffer)
	p.WritePath(path)
	n := C.file_size((*C.char)(p.Pointer()))
	p.Free()
	return int64(n), nil
}

func GetFileSizeAt(fd int, path string, followLinks bool) (int64, error) {
	// TODO: support excluding files and handling symlinks
	const (
		SYS_FSTATAT   = 469
		SYS_FSTATAT64 = 470
	)
	const (
		AT_FDCWD            = -0x2
		AT_REMOVEDIR        = 0x80
		AT_SYMLINK_FOLLOW   = 0x40
		AT_SYMLINK_NOFOLLOW = 0x20
	)

	p := bufPool.Get().(*buffer)
	p.WritePath(path)

	// var flags int
	// if !followLinks {
	// 	flags = AT_SYMLINK_NOFOLLOW
	// }

	var stat syscall.Stat_t
	_, _, e1 := syscall.Syscall6(
		SYS_FSTATAT64,
		uintptr(fd),
		// uintptr(unsafe.Pointer(&b[0])),
		uintptr(p.Pointer()),
		uintptr(unsafe.Pointer(&stat)),
		uintptr(AT_SYMLINK_NOFOLLOW), // WARN
		0,
		0,
	)
	p.Free()
	if e1 != 0 {
		return 0, os.NewSyscallError("fstatat64", e1)
	}
	return stat.Size, nil
}
