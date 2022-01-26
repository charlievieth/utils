//go:build appengine || (!linux && !darwin && !freebsd && !openbsd && !netbsd)
// +build appengine !linux,!darwin,!freebsd,!openbsd,!netbsd

package main

import "github.com/charlievieth/utils/fastwalk"

func GetFileSize(_ string, de fastwalk.DirEntry) (int64, error) {
	var size int64
	fi, err := de.Info()
	if err == nil {
		size = fi.Size()
	}
	return size, err
}
