package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"text/tabwriter"
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

type Line struct {
	s string
	n int
}

type byCount []Line

func (b byCount) Len() int           { return len(b) }
func (b byCount) Less(i, j int) bool { return b[i].n < b[j].n }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type byName []Line

func (b byName) Len() int           { return len(b) }
func (b byName) Less(i, j int) bool { return b[i].s < b[j].s }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func ReadStdin() ([]Line, error) {
	in := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	m := make(map[string]int, 128)

	var err error
	for err == nil {
		var b []byte
		b, err = in.ReadBytes('\n')
		if b = bytes.TrimSpace(b); len(b) != 0 {
			m[string(b)]++
		}
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	lines := make([]Line, 0, len(m))
	for s, n := range m {
		lines = append(lines, Line{s: s, n: n})
	}
	sort.Sort(byName(lines))
	sort.Stable(byCount(lines))

	return lines, nil
}

func main() {
	lines, err := ReadStdin()
	if err != nil {
		Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, l := range lines {
		b = b[:0]
		b = strconv.AppendInt(b, int64(l.n), 10)
		b = append(b, ':')
		b = append(b, '\t')
		b = append(b, l.s...)
		b = append(b, '\n')
		if _, err := w.Write(b); err != nil {
			Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		Fatal(err)
	}
}

func Fatal(err interface{}) {
	if err != nil {
		var s string
		if _, file, line, ok := runtime.Caller(1); ok && file != "" {
			s = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
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
