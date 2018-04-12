package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
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
	if len(frag) > 1 {
		r.buf = append(r.buf, frag[:len(frag)-1]...)
	}
	return r.buf, err
}

func ReadLines(r *Reader, delim byte) error {
	out := bufio.NewWriterSize(os.Stdout, 32*1024)
	scratch := make([]byte, 0, 64)
	var err error
	for {
		b, er := r.ReadBytes(delim)
		if n := int64(len(b)); n != 0 {
			scratch = strconv.AppendInt(scratch[:0], n, 10)
			_, ew := out.Write(append(scratch, '\n'))
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	if err != nil && err != io.EOF {
		out.Flush()
		return err
	}
	return out.Flush()
}

var NullTerminate bool

func parseFlags() {
	flag.BoolVar(&NullTerminate, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	flag.Parse()
}

func parseStdin(delim byte) error {
	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	return ReadLines(&r, delim)
}

func parseFiles(delim byte, names ...string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	setErr := func(err error) {
		if err != nil {
			mu.Lock()
			if first == nil {
				first = err
			}
			mu.Unlock()
		}
	}

	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			f, err := os.Open(name)
			if err != nil {
				setErr(err)
				return
			}
			defer f.Close()
			r := Reader{
				b:   bufio.NewReaderSize(f, 64*1024),
				buf: make([]byte, 128),
			}
			if err := ReadLines(&r, delim); err != nil {
				setErr(err)
			}
		}(name)
	}
	wg.Wait()

	return first
}

func main() {
	parseFlags()
	delim := byte('\n')
	if NullTerminate {
		delim = 0
	}
	var err error
	if flag.NArg() == 0 {
		err = parseStdin(delim)
	} else {
		err = parseFiles(delim, flag.Args()...)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
