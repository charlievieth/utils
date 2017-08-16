package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

func Ext(path []byte) []byte {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return nil
}

func main() {
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 4096),
		buf: make([]byte, 0, 128),
	}
	var buf []byte
	var err error
	for {
		buf, err = r.ReadBytes('\n')
		if err != nil {
			break
		}
		if _, err := os.Stdout.Write(Ext(buf)); err != nil {
			fmt.Fprintf(os.Stderr, "writing: %s\n", err)
			return
		}
	}
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "reading: %s\n", err)
		return
	}
	if _, err := os.Stdout.Write(Ext(buf)); err != nil {
		fmt.Fprintf(os.Stderr, "writing: %s\n", err)
		return
	}
	return
}
