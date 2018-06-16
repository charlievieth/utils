package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
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

func (r *Reader) ReadLines() ([]string, error) {
	var buf []byte
	var err error
	lines := make([]string, 0, 512)
	for {
		buf, err = r.ReadBytes('\n')
		if err != nil {
			break
		}
		if len(buf) != 0 {
			lines = append(lines, string(buf[:len(buf)-1]))
		}
	}
	if err != io.EOF {
		return nil, err
	}
	return append(lines, string(buf)), nil
}

func main() {
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 32*1024),
		buf: make([]byte, 0, 4096),
	}
	lines, err := r.ReadLines()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	sort.Strings(lines)
	out := bufio.NewWriterSize(os.Stdout, 32*1024)
	for _, s := range lines {
		if _, err := out.WriteString(s); err != nil {
			fmt.Fprintf(os.Stderr, "writing: %s\n", err)
			return
		}
	}
	if err := out.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flushing: %s\n", err)
		return
	}
	return
}
