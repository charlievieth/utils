package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"
)

type byLen []string

func (b byLen) Len() int           { return len(b) }
func (b byLen) Less(i, j int) bool { return len(b[i]) < len(b[j]) }
func (b byLen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func Fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

var printTime bool

func init() {
	flag.BoolVar(&printTime, "time", false, "print runtime")
}

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

func main() {
	flag.Parse()
	startTime := time.Now()

	in := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	lines := make([]string, 0, 128)
	var err error
	for err == nil {
		var b []byte
		b, err = in.ReadBytes('\n')
		if b = bytes.TrimSpace(b); len(b) != 0 {
			lines = append(lines, string(b))
		}
	}
	if err != nil && err != io.EOF {
		Fatalf("reading stdin: %s\n", err)
	}

	sortTime := time.Now()

	sort.Strings(lines)
	sort.Stable(byLen(lines))

	writeTime := time.Now()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, s := range lines {
		b = strconv.AppendInt(b[:0], int64(len(s)), 10)
		b = append(b, ' ')
		b = append(b, s...)
		b = append(b, '\n')
		if _, err := w.Write(b); err != nil {
			Fatalf("write: %s\n", err)
		}
	}

	if printTime {
		end := time.Now()
		fmt.Fprint(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "read\t%s\n", sortTime.Sub(startTime))
		fmt.Fprintf(os.Stderr, "sort\t%s\n", writeTime.Sub(sortTime))
		fmt.Fprintf(os.Stderr, "write\t%s\n", end.Sub(writeTime))
		fmt.Fprintf(os.Stderr, "total\t%s\n", end.Sub(startTime))
	}
}
