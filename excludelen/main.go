package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: [MAX_LINE_LENGTH]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
}

type Reader struct {
	b   *bufio.Reader
	buf []byte
	out []byte
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

// Returns the number held by r excluding ANSI escape sequences.  That is if the
// content of r.buf is written to a terminal this would be the number of visible
// characters printed.
func (r *Reader) PrintLen() int {
	count := 0
	o := 0
	n := len(r.buf)
	for {
		start, end := findIndexANSI(r.buf[o:])
		if start == -1 {
			break
		}
		if start > 0 {
			count += start
		}
		if end < n {
			o += end
		} else {
			o = n
			break
		}
	}
	return count + n - o
}

// N.B. only here for testing
func (r *Reader) stripANSI(out []byte) []byte {
	out = out[:0]
	buf := r.buf
	for {
		start, end := findIndexANSI(buf)
		if start == -1 {
			break
		}
		if start > 0 {
			out = append(out, buf[:start]...)
		}
		if end < len(buf) {
			buf = buf[end:]
		} else {
			buf = buf[len(buf):]
			break
		}
	}
	out = append(out, buf...)
	return out
}

func findIndexANSI(b []byte) (int, int) {
	// Pattern: \x1b\[[0-?]*[ -/]*[@-~]
	const minLen = 2 // "\\[[@-~]"

	start := bytes.IndexByte(b, '\x1b')
	if start == -1 || len(b)-start < minLen || b[start+1]-'@' > '_'-'@' {
		return -1, -1
	}

	n := start + 2 // ESC + second byte [@-_]

	// parameter bytes
	for ; n < len(b) && b[n]-'0' <= '?'-'0'; n++ {
	}
	// intermediate bytes
	for ; n < len(b) && b[n]-' ' <= '/'-' '; n++ {
	}
	// final byte
	if n < len(b) && b[n]-'@' <= '~'-'@' {
		return start, n + 1
	}
	return -1, -1
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "error: too many flags: %s\n", flag.Args())
		flag.Usage()
	}
	max, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing MAX_LINE_LENGTH: %s\n", err)
		flag.Usage()
	}
	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	for {
		b, e := r.ReadBytes('\n')
		if n := r.PrintLen(); n != 0 && n <= max {
			if _, err := os.Stdout.Write(b); err != nil && e == nil {
				e = err
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
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
