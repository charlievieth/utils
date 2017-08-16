package main

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"runtime"
)

func main() {
	if len(os.Args) < 2 {
		Fatalf("usage: <gosha> <filepaths>")
	}
	h := sha256.New()
	for _, name := range os.Args[1:] {
		b, err := HashFile(name, h)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error (%s): %s\n", name, err)
			continue
		}
		fmt.Fprintf(os.Stdout, "%x  %s\n", b, name)
	}
}

func HashFile(name string, h hash.Hash) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h.Reset()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func Fatalf(format string, a ...interface{}) {
	Fatal(fmt.Sprintf(format, a...))
}

func Fatal(err interface{}) {
	var s string
	if _, file, line, ok := runtime.Caller(1); ok {
		s = fmt.Sprintf("%s:%d", file, line)
	}
	if err != nil {
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
