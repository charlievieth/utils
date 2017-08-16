package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

var ZeroDelim bool

func init() {
	flag.BoolVar(&ZeroDelim, "0", false, "Zero delim")
}

func main() {
	flag.Parse()
	var delim byte
	if !ZeroDelim {
		delim = '\n'
	}
	fmt.Println("ZeroDelim:", ZeroDelim, delim)
}

func Fatal(err interface{}) {
	if err != nil {
		var s string
		if _, file, line, ok := runtime.Caller(1); ok && file != "" {
			s = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
