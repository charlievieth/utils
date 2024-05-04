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
	printTime := flag.Bool("time", false, "print runtime")
	printLength := flag.Bool("length", false, "print the length of each line")
	trim := flag.Bool("trim", false, "trim leading and trailing whitespace")
	flag.Parse()
	startTime := time.Now()

	in := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	lines := make([]string, 0, 128)
	doTrim := *trim
	var err error
	for err == nil {
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
	if err != nil {
		Fatalf("reading stdin: %s\n", err)
	}

	sortTime := time.Now()

	slices.SortFunc(lines, func(s1, s2 string) int {
		n1 := len(s1)
		n2 := len(s2)
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
		return cmp.Compare(s1, s2)
	})

	writeTime := time.Now()

	if *printLength {
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

	if *printTime {
		end := time.Now()
		fmt.Fprint(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "read\t%s\n", sortTime.Sub(startTime))
		fmt.Fprintf(os.Stderr, "sort\t%s\n", writeTime.Sub(sortTime))
		fmt.Fprintf(os.Stderr, "write\t%s\n", end.Sub(writeTime))
		fmt.Fprintf(os.Stderr, "total\t%s\n", end.Sub(startTime))
	}
}
