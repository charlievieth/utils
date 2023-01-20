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
	"strconv"
	"text/tabwriter"

	"github.com/charlievieth/num"
	"golang.org/x/exp/slices"
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
	if n := len(frag); n != 0 && frag[n-1] == delim {
		frag = frag[:n-1]
	}
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

type Line struct {
	S string
	N int
}

// TODO: remove ignore case option
func ReadLines(r *Reader, delim byte, ignoreCase, reverseOrder bool) ([]Line, error) {
	m := make(map[string]*int, 128)

	var err error
	for {
		// TOOD: trim space ???
		b, e := r.ReadBytes(delim)
		if len(b) != 0 {
			if ignoreCase {
				b = bytes.ToLower(b)
			}
			p := m[string(b)]
			if p == nil {
				p = new(int)
				m[string(b)] = p
			}
			*p++
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if err != nil {
		return nil, err
	}

	lines := make([]Line, 0, len(m))
	for s, n := range m {
		lines = append(lines, Line{S: s, N: *n})
	}

	if reverseOrder {
		slices.SortFunc(lines, func(a, b Line) bool {
			return a.N > b.N || (a.N == b.N && a.S < b.S)
		})
	} else {
		slices.SortFunc(lines, func(a, b Line) bool {
			return a.N < b.N || (a.N == b.N && a.S < b.S)
		})
	}

	return lines, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTION]...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	nullTerminate := flag.Bool("0", false, "line delimiter is NUL, not newline")
	reverseOrder := flag.Bool("r", false, "reverse frequency sort order")
	printThousandsSep := flag.Bool("n", false, "print numbers with thousands separators")
	caseInsensitive := flag.Bool("i", false, "sort names case-insensitively")
	flag.Parse()

	// parseFlags()
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 64*1024),
		buf: make([]byte, 128),
	}
	delim := byte('\n')
	if *nullTerminate {
		delim = 0
	}
	lines, err := ReadLines(&r, delim, *caseInsensitive, *reverseOrder)
	if err != nil {
		Fatal(err)
	}
	r = Reader{} // clear reference

	thousands := *printThousandsSep

	bw := bufio.NewWriter(os.Stdout)
	w := tabwriter.NewWriter(bw, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128) // format buffer

	for _, l := range lines {
		b = b[:0]
		if thousands {
			b = append(b, num.FormatInt(int64(l.N))...)
		} else {
			b = strconv.AppendInt(b, int64(l.N), 10)
		}
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
	if err := bw.Flush(); err != nil {
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
