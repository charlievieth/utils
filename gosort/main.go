package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
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
	Raw   string
	Lower string
}

func (r *Reader) ReadLines(ignoreCase bool) ([]Line, error) {
	var buf []byte
	var err error
	lines := make([]Line, 0, 64)
	for {
		buf, err = r.ReadBytes('\n')
		if len(buf) != 0 {
			ln := Line{Raw: string(buf)}
			if ignoreCase {
				ln.Lower = strings.ToLower(ln.Raw)
			} else {
				ln.Lower = ln.Raw
			}
			lines = append(lines, ln)
		}
		if err != nil {
			break
		}
	}
	if err != io.EOF {
		return nil, err
	}
	return lines, nil
}

func realMain() error {
	ignoreCase := flag.Bool("f", false, "ignore case when sorting")
	flag.Parse()
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 32*1024),
		buf: make([]byte, 0, 4096),
	}
	lines, err := r.ReadLines(*ignoreCase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return err
	}
	out := bufio.NewWriterSize(os.Stdout, 32*1024)
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Lower < lines[j].Lower
	})
	for _, x := range lines {
		if _, err := out.WriteString(x.Raw); err != nil {
			fmt.Fprintf(os.Stderr, "writing: %s\n", err)
			return err
		}
	}
	if err := out.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flushing: %s\n", err)
		return err
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
