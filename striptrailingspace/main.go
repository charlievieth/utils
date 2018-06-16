package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		b:   bufio.NewReader(r),
		buf: make([]byte, 0, 128),
	}
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

func isSpace(r rune) bool {
	return r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r' ||
		r == ' ' || r == 0x85 || r == 0xA0
}

func Strip(rd io.Reader, wr io.Writer) error {
	r := NewReader(rd)
	w := bufio.NewWriter(wr)
	var err error
	for {
		b, e := r.ReadBytes('\n')
		if len(b) != 0 {
			b = append(bytes.TrimRightFunc(b, isSpace), '\n')
			if _, ew := w.Write(b); ew != nil && e == nil {
				e = ew
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
		return err
	}
	return w.Flush()
}

func StripInPlace(name string) error {
	path, err := filepath.Abs(name)
	if err != nil {
		return nil
	}
	fi, err := os.Open(name)
	if err != nil {
		return err
	}
	defer fi.Close()
	fo, err := ioutil.TempFile(filepath.Dir(path), "strip_")
	if err != nil {
		return err
	}
	err = Strip(fi, fo)
	fo.Close()
	if err != nil {
		os.Remove(fo.Name())
		return err
	}
	fi.Close()
	if err := os.Rename(fo.Name(), path); err != nil {
		os.Remove(fo.Name())
		return err
	}
	return nil
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "USAGE: FILENAME...")
		os.Exit(1)
	}
	for _, name := range os.Args[1:] {
		if err := StripInPlace(name); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}
