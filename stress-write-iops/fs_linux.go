//go:build linux
// +build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func Fallocate(filename string, size int64) error {
	if size < 0 {
		return errors.New("fallocate: negative size")
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Fallocate(int(f.Fd()), 0, 0, size); err != nil {
		return fmt.Errorf("fallocate: %w", err)
	}
	return f.Close()
}
