package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
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

func Ext(path []byte) []byte {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return nil
}

var NullTerminate bool

func parseFlags() {
	flag.BoolVar(&NullTerminate, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	flag.Parse()
}

func realMain() error {
	parseFlags()
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 4096),
		buf: make([]byte, 0, 128),
	}
	var buf []byte
	var err error
	if NullTerminate {
		for {
			buf, err = r.ReadBytes(0)
			if err != nil {
				break
			}
			buf[len(buf)-1] = '\n'
			if _, err := os.Stdout.Write(Ext(buf)); err != nil {
				return fmt.Errorf("writing: %s\n", err)
			}
		}
	} else {
		for {
			buf, err = r.ReadBytes('\n')
			if err != nil {
				break
			}
			if _, err := os.Stdout.Write(Ext(buf)); err != nil {
				return fmt.Errorf("writing: %s\n", err)
			}
		}
	}
	if err != io.EOF {
		return fmt.Errorf("reading: %s\n", err)
	}
	if _, err := os.Stdout.Write(Ext(buf)); err != nil {
		return fmt.Errorf("writing: %s\n", err)
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return
}
