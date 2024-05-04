package main

import (
	"bufio"
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
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
	if n := len(frag); n > 0 && frag[n-1] == delim {
		frag = frag[:n-1]
	}
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

func main() {
	stableSort := flag.Bool("stable", false, "use a stable sort")
	sortStrings := flag.Bool("comp", false, "sort by length and tie-break on string")
	printTime := flag.Bool("time", false, "print runtime")
	printLength := flag.Bool("length", false, "print the length of each line")
	trim := flag.Bool("trim", false, "trim leading and trailing whitespace")
	flag.Parse()

	if *stableSort && *sortStrings {
		Fatalf("cannot specify both the '-stable' and '-comp' flags")
	}

	in := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 256*1024),
		buf: make([]byte, 128),
	}
	lines := make([]string, 0, 128)

	files := flag.Args()
	if len(files) == 0 {
		files = append(files, "-")
	}

	startTime := time.Now()
	doTrim := *trim
	var err error
	for _, name := range files {
		var f *os.File
		if name == "-" {
			f = os.Stdin
		} else {
			f, err = os.Open(name)
			if err != nil {
				Fatalf("%s: %s\n", name, err)
			}
		}
		in.b.Reset(f)
		for {
			b, e := in.ReadBytes('\n')
			if doTrim {
				b = bytes.TrimSpace(b)
			}
			if len(b) != 0 {
				lines = append(lines, string(b))
			}
			if e != nil {
				if e != io.EOF {
					err = e
				}
				break
			}
		}
		if f != os.Stdin {
			f.Close()
		}
		if err != nil {
			Fatalf("reading %s: %s\n", err)
		}
	}

	sortTime := time.Now()
	switch {
	case *stableSort:
		slices.SortStableFunc(lines, func(s1, s2 string) int {
			return len(s1) - len(s2)
		})
	case *sortStrings:
		slices.SortFunc(lines, func(s1, s2 string) int {
			n := len(s1) - len(s2)
			if n != 0 {
				return n
			}
			return cmp.Compare(s1, s2)
		})
	default:
		slices.SortFunc(lines, func(s1, s2 string) int {
			return len(s1) - len(s2)
		})
	}

	writeTime := time.Now()
	if len(lines) > 0 {
		if *printLength {
			n := len(strconv.Itoa(len(lines[len(lines)-1])))
			n++
			if n < 4 {
				n = n
			}
			for _, s := range lines {
				if _, err := fmt.Printf("%*d  %s\n", n, len(s), s); err != nil {
					Fatalf("write: %v\n", err)
				}
			}
		} else {
			w := bufio.NewWriter(os.Stdout)
			for _, s := range lines {
				w.WriteString(s)
				w.WriteByte('\n')
			}
			if err := w.Flush(); err != nil {
				Fatalf("write: %s\n", err)
			}
		}
	}

	if *printTime {
		end := time.Now()
		fmt.Fprint(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "read\t%s\n", sortTime.Sub(startTime))
		fmt.Fprintf(os.Stderr, "sort\t%s\n", writeTime.Sub(sortTime))
		fmt.Fprintf(os.Stderr, "write\t%s\n", end.Sub(writeTime))
		fmt.Fprintf(os.Stderr, "total\t%s\n", end.Sub(startTime))
	}
}
