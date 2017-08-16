package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func NewReader(rd io.Reader, size int) *Reader {
	return &Reader{
		b:   bufio.NewReaderSize(rd, size),
		buf: make([]byte, 0, size),
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
	return r.buf, err
}

func NullTerminate() error {
	r := bufio.NewReaderSize(os.Stdin, 4096)

	var buf []byte
	var err error
	for {
		buf, err = r.ReadBytes('\n')
		if err != nil {
			break
		}
		if len(buf) != 0 {
			// fmt.Fprint(os.Stderr, "out: "+string(buf))
			// buf[len(buf)-1] = 0
			_, err = os.Stdout.Write(buf[:len(buf)-1])
			os.Stdout.Write([]byte{0})
		}
	}
	if err != io.EOF {
		return err
	}
	err = nil
	if len(buf) != 0 {
		// fmt.Fprint(os.Stderr, "out: "+string(buf))
		// buf[len(buf)-1] = 0
		// _, err = os.Stdout.Write(buf)
		_, err = os.Stdout.Write(buf[:len(buf)-1])
		os.Stdout.Write([]byte{0})
	}
	return err
}

func main() {
	if err := NullTerminate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
