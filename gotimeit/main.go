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
	"text/tabwriter"
	"time"
)

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func NewReader(b *bufio.Reader) *Reader {
	return &Reader{
		b:   b,
		buf: make([]byte, 128),
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
	if len(r.buf) != 0 {
		r.buf = r.buf[:len(r.buf)-1]
	}
	return r.buf, err
}

type Line struct {
	Time     time.Time
	Line     []byte
	Duration time.Duration
}

type ByDuration []Line

func (b ByDuration) Len() int           { return len(b) }
func (b ByDuration) Less(i, j int) bool { return b[i].Duration < b[j].Duration }
func (b ByDuration) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func main() {
	flag.Usage = func() {
		const msg = "Usage: %[1]s [OPTION]...\n" +
			"Append timestamps to each line of standard input.\n"
		fmt.Fprintf(flag.CommandLine.Output(), msg, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	printLines := flag.Bool("n", false, "Print line numbers")
	sortDur := flag.Bool("s", false, "Print lines sorted by duration (implies -n)")
	flag.Parse()
	if *sortDur {
		*printLines = true
	}

	lines := make([]Line, 0, 8)
	r := NewReader(bufio.NewReader(os.Stdin))
	var err error
	for {
		b, e := r.ReadBytes('\n')
		lines = append(lines, Line{
			Time: time.Now(),
			Line: append([]byte(nil), b...),
		})
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if err != nil {
		Fatal(err)
	}
	if len(lines) == 0 {
		return
	}

	start := lines[0].Time
	for i := 1; i < len(lines); i++ {
		lines[i].Duration = lines[i].Time.Sub(lines[i-1].Time)
	}
	if *sortDur {
		sort.Stable(sort.Reverse(ByDuration(lines)))
	}

	const minWidth = len("0.000001")
	var buf bytes.Buffer

	for i, ll := range lines {
		if *printLines {
			fmt.Fprintf(&buf, "%d\t", i+1)
		}
		fmt.Fprintf(&buf, "%.6f\t", ll.Time.Sub(start).Seconds())
		fmt.Fprintf(&buf, "%.6f\t", ll.Duration.Seconds())
		buf.WriteByte(tabwriter.Escape)
		buf.Write(ll.Line)
		buf.WriteByte(tabwriter.Escape)
		buf.WriteByte('\n')
	}

	w := tabwriter.NewWriter(os.Stdout, minWidth, 0, 2, ' ', tabwriter.StripEscape)
	buf.WriteTo(w)
	if err := w.Flush(); err != nil {
		Fatal(err)
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
