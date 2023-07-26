//go:build (linux && !appengine) || darwin || freebsd || openbsd || netbsd
// +build linux,!appengine darwin freebsd openbsd netbsd

package main

import (
	"syscall"

	"github.com/charlievieth/fastwalk"
)

func GetFileSize(path string, _ fastwalk.DirEntry) (int64, error) {
	// CEV: bad name, but I'm too lazy to rename the other poorly
	// named FileSize
	var stat syscall.Stat_t
	for {
		err := syscall.Lstat(path, &stat)
		if err != syscall.EINTR {
			return int64(stat.Size), err
		}
	}
}
