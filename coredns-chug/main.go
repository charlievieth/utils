package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// includes delim
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

type LogLine struct {
	Time     time.Time
	Duration time.Duration
	Line     string
}

func ParseLogLine(s string) (*LogLine, error) {
	n := strings.IndexByte(s, ' ')
	if n == -1 {
		return nil, errors.New("missing timestamp")
	}
	t, err := time.Parse(time.RFC3339Nano, s[:n])
	if err != nil {
		return nil, err
	}
	n = strings.LastIndexByte(s, ' ')
	if n == -1 {
		return nil, errors.New("missing duration")
	}
	d, err := time.ParseDuration(s[n+1:])
	if err != nil {
		return nil, err
	}
	ll := &LogLine{
		Time:     t,
		Duration: d,
		Line:     s,
	}
	return ll, nil
}

func StreamLines(rd io.Reader) ([]*LogLine, error) {
	r := NewReader(bufio.NewReader(rd))
	var lines []*LogLine
	var err error
	for {
		b, e := r.ReadBytes('\n')
		b = bytes.TrimSpace(b)
		if len(b) != 0 {
			line := string(b)
			ll, err := ParseLogLine(line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s: %s\n", err, line)
			} else {
				lines = append(lines, ll)
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if len(lines) == 0 && err == nil {
		err = errors.New("no log lines")
	}
	return lines, err
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: [OPTIONS] FILENAME\n",
			filepath.Dir(os.Args[0]))
		flag.PrintDefaults()
	}
	var minDuration time.Duration
	flag.DurationVar(&minDuration, "min_dur", 0, "Minimum duration")
	flag.Parse()

	if flag.NArg() > 2 {
		flag.Usage()
		os.Exit(1)
	}
	var rd io.Reader = os.Stdin
	if flag.NArg() == 1 && flag.Arg(0) != "-" {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		defer f.Close()
		rd = f
	}

	lines, err := StreamLines(rd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if minDuration > 0 {
		a := lines[:0]
		for _, ll := range lines {
			if ll.Duration >= minDuration {
				a = append(a, ll)
			}
		}
		lines = a
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Time.Before(lines[j].Time)
	})
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Duration < lines[j].Duration
	})
	for _, ll := range lines {
		fmt.Println(ll.Line)
	}
}
