package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func (r *Reader) ReadBytes(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	r.buf = r.buf[:0]
	for {
		var e error
		frag, e = r.b.ReadSlice(delim)
		if e == nil { // got final fragment
			break
		}
		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}
		r.buf = append(r.buf, frag...)
	}
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

func VolumeName(path []byte) []byte {
	return path[:volumeNameLen(path)]
}

func Base(path []byte) []byte {
	if len(path) == 0 {
		return nil
	}
	// Strip trailing slashes.
	for len(path) > 0 && os.IsPathSeparator(path[len(path)-1]) {
		path = path[0 : len(path)-1]
	}
	// Throw away volume name
	path = path[len(VolumeName(path)):]
	// Find the last element
	i := len(path) - 1
	for i >= 0 && !os.IsPathSeparator(path[i]) {
		i--
	}
	if i >= 0 {
		path = path[i+1:]
	}
	// If empty now, it had only slashes.
	if len(path) == 0 {
		return []byte{filepath.Separator}
	}
	return path
}

var ZeroDelim bool
var ZeroTerm bool

func parseFlags() {
	flag.BoolVar(&ZeroDelim, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	flag.BoolVar(&ZeroTerm, "z", false,
		"End each output line with NUL ('\\0'), not newline")
	flag.Parse()
}

func main() {
	parseFlags()
	delim := byte('\n')
	if ZeroDelim {
		delim = 0
	}
	eol := byte('\n')
	if ZeroTerm {
		eol = 0
	}
	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	var err error
	for err == nil {
		var b []byte
		b, err = r.ReadBytes(delim)
		if len(b) <= 1 {
			continue
		}
		b[len(b)-1] = eol // replace delim with newline
		if b = Base(b); len(b) != 0 {
			_, e := os.Stdout.Write(b)
			if e != nil && err == nil {
				err = e
			}
		}
	}
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
