//go:build !linux
// +build !linux

package main

import (
	"errors"
	"io"
	"os"
)

type inifiniteReader struct{}

func (inifiniteReader) Read(p []byte) (int, error) {
	return len(p), nil
}

// Crappy Fallocate for non-Linux systems
func Fallocate(filename string, size int64) error {
	if size < 0 {
		return errors.New("fallocate: negative size")
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil {
		if fi.Size() < size {
			if err := f.Truncate(size); err != nil {
				return err
			}
			return f.Close()
		}
	}

	r := io.LimitReader(inifiniteReader{}, size)
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	return f.Close()
}
