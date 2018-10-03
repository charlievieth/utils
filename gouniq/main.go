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

func isSpace(r byte) bool {
	return r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r' ||
		r == ' ' || r == 0x85 || r == 0xA0
}

func trimSpace(s []byte) []byte {
	i := 0
	for ; i < len(s) && isSpace(s[i]); i++ {
	}
	s = s[i:]
	i = len(s) - 1
	for ; i >= 0 && isSpace(s[i]); i-- {
	}
	return s[:i+1]
}

func UniqLines(in io.Reader, delim byte) ([]string, error) {
	r := Reader{
		b:   bufio.NewReader(in),
		buf: make([]byte, 128),
	}
	seen := make(map[string]struct{})
	lines := make([]string, 0, 64)
	var err error
	for {
		b, e := r.ReadBytes(delim)
		b = trimSpace(b)
		if len(b) != 0 {
			if _, ok := seen[string(b)]; !ok {
				seen[string(b)] = struct{}{}
				lines = append(lines, string(b))
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	return lines, err
}

func StreamLines(in io.Reader, out io.Writer, delim byte) error {
	r := Reader{
		b:   bufio.NewReader(in),
		buf: make([]byte, 128),
	}
	seen := make(map[string]struct{})
	var err error
	for {
		b, er := r.ReadBytes(delim)
		b = trimSpace(b)
		if len(b) != 0 {
			if _, ok := seen[string(b)]; !ok {
				seen[string(b)] = struct{}{}
				_, ew := out.Write(append(b, '\n'))
				if ew != nil {
					if er == nil || er == io.EOF {
						er = ew
					}
				}
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
}

func processFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return StreamLines(f, os.Stdout, '\n')
}

func realMain() error {
	flag.Parse()
	if flag.NArg() == 0 {
		return StreamLines(os.Stdin, os.Stdout, '\n')
	}
	for _, name := range flag.Args() {
		if err := processFile(name); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
