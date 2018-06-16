package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

const (
	_  = iota
	kB = 1 << (10 * iota)
	mB
	gB
	tB
)

func main() {
	var (
		lines int64
		count int64
		err   error
	)
	buf := make([]byte, 1024*8)
	for {
		nr, er := os.Stdin.Read(buf)
		if nr > 0 {
			lines += int64(bytes.Count(buf[:nr], []byte{'\n'}))
			count += int64(nr)
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	fmt.Fprintf(w, "lines\t%d\n", lines)
	fmt.Fprintf(w, "bytes\t%d\n", count)

	if kb := float64(count) / kB; kb > 0.1 {
		fmt.Fprintf(w, "kB\t%.2f\n", kb)
	}
	if mb := float64(count) / mB; mb > 0.1 {
		fmt.Fprintf(w, "mB\t%.2f\n", mb)
	}
	if gb := float64(count) / gB; gb > 0.1 {
		fmt.Fprintf(w, "gB\t%.2f\n", gb)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
