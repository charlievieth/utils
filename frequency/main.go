package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	S string
	L string // lower
	N int
}

type byName []Line

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].L < b[j].L }

type byCount []Line

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].N < b[j].N }

type byCountReverse []Line

func (b byCountReverse) Len() int           { return len(b) }
func (b byCountReverse) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCountReverse) Less(i, j int) bool { return b[i].N >= b[j].N }

type byNameCount []Line

func (b byNameCount) Len() int      { return len(b) }
func (b byNameCount) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byNameCount) Less(i, j int) bool {
	return b[i].N < b[j].N || (b[i].N == b[j].N && b[i].S < b[j].S)
}

func ReadLines(r *Reader, delim byte, ignoreCase, reverseOrder bool) ([]Line, error) {
	m := make(map[string]int, 128)

	var err error
	for err == nil {
		var b []byte
		b, err = r.ReadBytes(delim)
		if b = bytes.TrimSpace(b); len(b) != 0 {
			m[string(b)]++
		}
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	lines := make([]Line, 0, len(m))
	if ignoreCase {
		for s, n := range m {
			lines = append(lines, Line{S: s, L: strings.ToLower(s), N: n})
		}
	} else {
		for s, n := range m {
			lines = append(lines, Line{S: s, N: n})
		}
	}
	if ignoreCase {
		sort.Sort(byName(lines))
		if reverseOrder {
			sort.Stable(byCountReverse(lines))
		} else {
			sort.Stable(byCount(lines))
		}
	} else {
		if reverseOrder {
			sort.Sort(byName(lines))
			sort.Stable(byCountReverse(lines))
		} else {
			sort.Sort(byNameCount(lines))
		}
	}
	return lines, nil
}

var (
	NullTerminate   bool
	CaseInsensitive bool
	ReverseOrder    bool
)

func parseFlags() {
	flag.BoolVar(&NullTerminate, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	flag.BoolVar(&CaseInsensitive, "case", false,
		"Sort names case-insensitively")
	flag.BoolVar(&ReverseOrder, "r", false,
		"Reverse frequency sort order.")
	flag.Parse()
}

func main() {
	parseFlags()
	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	delim := byte('\n')
	if NullTerminate {
		delim = 0
	}
	lines, err := ReadLines(&r, delim, CaseInsensitive, ReverseOrder)
	if err != nil {
		Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, l := range lines {
		b = b[:0]
		b = strconv.AppendInt(b, int64(l.N), 10)
		b = append(b, ':')
		b = append(b, '\t')
		b = append(b, l.S...)
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
