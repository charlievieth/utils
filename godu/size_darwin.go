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
// static int64_t file_size(const char *restrict path) {
// 	struct stat st;
// 	if (unlikely(lstat(path, &st) != 0)) {
// 		return 0;
// 	}
// 	return (int64_t)st.st_size;
// }
import "C"

import (
	"unsafe"
)

// TODO: pass the file descriptor instead of the path
//
// Nasty hack to avoid using the libc_lstat64_trampoline, which is dog slow.
func GetFileSize(path string) (int64, error) {
	// TODO: we can also just call stat directly
	p := C.CString(path)
	n := C.file_size(p)
	C.free(unsafe.Pointer(p))
	return int64(n), nil
}
