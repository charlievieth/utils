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

	"github.com/charlievieth/reonce"
	"golang.org/x/term"
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
	Percent  float64
}

var (
	tab        = []byte{'\t'}
	ansiEscape = []byte("\x1b[")
	ansiRe     = reonce.New(`(?U)` + "\x1b" + `\[.*m`)
)

func hasANSI(b []byte) bool {
	return len(b) > 2 && bytes.HasPrefix(b, ansiEscape) ||
		bytes.Contains(b, ansiEscape)
}

func (l *Line) ApparentLength(tabwidth int) int {
	if tabwidth < 0 {
		tabwidth = 8
	}
	if hasANSI(l.Line) {
		ll := ansiRe.ReplaceAll(l.Line, nil)
		return len(ll) + (bytes.Count(ll, tab) * (tabwidth - 1))
	}
	return len(l.Line) + (bytes.Count(l.Line, tab) * (tabwidth - 1))
}

type ByDuration []Line

func (b ByDuration) Len() int           { return len(b) }
func (b ByDuration) Less(i, j int) bool { return b[i].Duration < b[j].Duration }
func (b ByDuration) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// WARN WARN WARN

// WARN: use this to calculate the "correct" line length based on P95 or something
// https://stackoverflow.com/questions/3738349/fast-algorithm-for-repeated-calculation-of-percentile

// An IntHeap is a min-heap of ints.
type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *IntHeap) Push(x any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(int))
}

func (h *IntHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// WARN WARN WARN

var isTerminal = term.IsTerminal(1)

type RGB struct {
	R, G, B uint8
}

func (r *RGB) Printf(format string, a ...any) (int, error) {
	n1, err := fmt.Printf("\x1b[38;2;%d;%d;%dm", r.R, r.G, r.B)
	if err != nil {
		return n1, err
	}
	n2, err := fmt.Printf(format, a...)
	if err != nil {
		return n1 + n2, err
	}
	n3, err := fmt.Print("\x1b[0m")
	return n1 + n2 + n3, err
}

func main() {
	flag.Usage = func() {
		const msg = "Usage: %[1]s [OPTION]...\n" +
			"Append timestamps to each line of standard input.\n"
		fmt.Fprintf(flag.CommandLine.Output(), msg, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	printLines := flag.Bool("n", false, "Print line numbers")
	sortDur := flag.Bool("s", false, "Print lines sorted by duration (implies -n)")
	printPercentage := flag.Bool("p", false, "Print percentage of time")
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
	dur := lines[len(lines)-1].Time.Sub(start)
	for i := 1; i < len(lines); i++ {
		d := lines[i].Time.Sub(lines[i-1].Time)
		lines[i].Duration = d
		lines[i].Percent = (float64(d) / float64(dur)) * 100
	}
	if *printPercentage {
		// max := 0
		const esc = string(tabwriter.Escape)
		max := 0
		for i := 0; i < len(lines) && max <= 100; i++ {
			n := len(lines[i].Line)
			if n > max {
				max = n
			}
		}
		if max <= 100 {
			max += 2
		} else {
			max = 80
		}
		fmt.Fprintln(os.Stderr, "max:", max)
		for _, ll := range lines {

			// fmt.Fprintf(w, "%s%s%s\t(%.2f%%)\n", esc, ll.Line, esc, ll.Percent)

			n := max - ll.ApparentLength(-1)
			if n > 2 {
				fmt.Printf("%s%*s(%.2f%%)\n", ll.Line, n, "", ll.Percent)
			} else {
				fmt.Printf("%s  (%.2f%%)\n", ll.Line, ll.Percent)
			}
		}
		// WARN WARN WARN WARN
		return // WARN
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
