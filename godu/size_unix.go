// +build linux,!appengine darwin freebsd openbsd netbsd

package main

import "syscall"

func GetFileSize(path string) (int64, error) {
	// CEV: bad name, but I'm too lazy to rename the other poorly
	// named FileSize
	var stat syscall.Stat_t
	err := syscall.Lstat(path, &stat)
	return stat.Size, err
}
