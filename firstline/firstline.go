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

// echos the first line of input to stderr
func main() {
	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	first := true
	var err error
	for err == nil {
		b, e := r.ReadBytes('\n')
		if len(b) != 0 {
			if first {
				first = false
				if _, err := os.Stderr.Write(b); err != nil {
					if e == nil || e == io.EOF {
						e = err
					}
				}
			}
			if _, err := os.Stdout.Write(b); err != nil {
				if e == nil || e == io.EOF {
					e = err
				}
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
