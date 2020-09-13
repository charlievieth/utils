// +build darwin

package main

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func BenchmarkGetFileSize_Fast(b *testing.B) {
	if _, err := fastSize("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fastSize("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go")
		}
	})
}

func BenchmarkGetFileSize(b *testing.B) {
	if _, err := GetFileSize("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetFileSize("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go")
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
	b.Skip("TODO: remove")
	if _, err := sizeSyscall("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sizeSyscall("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go")
		}
	})
}

func BenchmarkGetFileSize_OS(b *testing.B) {
	b.Skip("TODO: remove")
	if _, err := os.Lstat("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			os.Lstat("/Users/cvieth/go/src/github.com/charlievieth/utils/godu/size_darwin_test.go")
		}
	})
}

func BenchmarkGetFileSizeAt(b *testing.B) {
	fd, err := syscall.Open("/Users/cvieth/go/src/github.com/charlievieth/utils/godu", 0, 0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetFileSizeAt(fd, "size_darwin_test.go", false)
		}
	})
}

func fastSize(path string) (int64, error) {
	// WARN: use non 64 variant
	const SYS_STAT64 = 338

	p := bufPool.Get().(*[]byte)
	b := *p
	copy(b, path)
	b[len(path)] = 0

	var stat syscall.Stat_t
	_, _, e1 := syscall.Syscall(
		SYS_STAT64,
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(unsafe.Pointer(&stat)),
		0,
	)
	bufPool.Put(p)
	if e1 != 0 {
		return 0, os.NewSyscallError("stat64", e1)
	}
	return stat.Size, nil
}
