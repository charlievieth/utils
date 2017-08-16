package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
)

type byName [][]byte

func (b byName) Len() int           { return len(b) }
func (b byName) Less(i, j int) bool { return bytes.Compare(b[i], b[j]) < 0 }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func (r *Reader) ReadBytes(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	i := len(r.buf)
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
	b := r.buf[i:len(r.buf)]
	return b, err
}

func (r *Reader) ReadLines() ([][]byte, error) {
	var buf []byte
	var err error
	lines := make([][]byte, 0, 512)
	for {
		buf, err = r.ReadBytes('\n')
		if err != nil {
			break
		}
		lines = append(lines, buf)
	}
	if err != io.EOF {
		return nil, err
	}
	return append(lines, buf), nil
}

func main() {
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 4096),
		buf: make([]byte, 0, 4096),
	}
	lines, err := r.ReadLines()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	sort.Sort(byName(lines))
	for _, b := range lines {
		if _, err := os.Stdout.Write(b); err != nil {
			fmt.Fprintf(os.Stderr, "writing: %s\n", err)
			return
		}
	}
	return
}
