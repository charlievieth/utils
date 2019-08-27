// +build appengine !linux,!darwin,!freebsd,!openbsd,!netbsd

package main

import "os"

func GetFileSize(path string) (int64, error) {
	// CEV: bad name, but I'm too lazy to rename the other poorly
	// named FileSize
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}
