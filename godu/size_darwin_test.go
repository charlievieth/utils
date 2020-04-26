// +build darwin

package main

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func BenchmarkGetFileSize_Fast(b *testing.B) {
	if _, err := fastSize("main.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fastSize("main.go")
		}
	})
}

func BenchmarkGetFileSize(b *testing.B) {
	if _, err := GetFileSize("main.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetFileSize("main.go")
		}
	})
}

// here for testing
func sizeSyscall(path string) (int64, error) {
	// CEV: bad name, but I'm too lazy to rename the other poorly
	// named FileSize
	var stat syscall.Stat_t
	err := syscall.Lstat(path, &stat)
	return stat.Size, err
}

func BenchmarkGetFileSize_Syscall(b *testing.B) {
	if _, err := sizeSyscall("main.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sizeSyscall("main.go")
		}
	})
}

func BenchmarkGetFileSize_OS(b *testing.B) {
	if _, err := os.Lstat("main.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			os.Lstat("main.go")
		}
	})
}

func fastSize(path string) (int64, error) {
	const SYS_STAT64 = 338

	a := make([]byte, len(path)+1)
	copy(a, path)
	p0 := &a[0]
	// p0, err := syscall.BytePtrFromString(path)
	// if err != nil {
	// 	Fatal(err)
	// }
	var stat syscall.Stat_t
	_, _, e1 := syscall.Syscall(SYS_STAT64, uintptr(unsafe.Pointer(p0)), uintptr(unsafe.Pointer(&stat)), 0)
	if e1 != 0 {
		return 0, e1
	}
	return stat.Size, nil
}
