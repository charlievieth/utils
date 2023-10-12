package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
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

func StreamLines(in io.Reader, out io.Writer, delim byte, ignoreSpace bool) error {
	const bufsz = 64 * 1024
	r := Reader{
		b:   bufio.NewReaderSize(in, bufsz),
		buf: make([]byte, 128),
	}
	w := bufio.NewWriterSize(out, bufsz)
	seen := make(map[string]struct{})
	var err error
	for {
		b, er := r.ReadBytes(delim)
		if ignoreSpace {
			b = trimSpace(b)
		}
		if len(b) != 0 {
			if _, ok := seen[string(b)]; !ok {
				seen[string(b)] = struct{}{}
				_, ew := w.Write(append(b, '\n'))
				if ew != nil {
					if er == nil || er == io.EOF {
						er = ew
					}
				}
			}
		}
		if er != nil {
			err = er
			break
		}
	}
	if err != io.EOF {
		return err
	}
	return w.Flush()
}

func processFile(name string, delim byte, ignoreSpace bool) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return StreamLines(f, os.Stdout, delim, ignoreSpace)
}

func realMain() error {
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage %s: [FILE]...\n"+
			"Print unique lines from [FILE]... or STDIN in the order they are received.\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	nullDelim := flag.Bool("z", false, "line delimiter is NUL, not newline")
	stripSpace := flag.Bool("s", false, "strip trailing and leading whitespace")
	flag.Parse()

	delim := byte('\n')
	if *nullDelim {
		delim = 0
	}
	if flag.NArg() == 0 {
		return StreamLines(os.Stdin, os.Stdout, delim, *stripSpace)
	}
	for _, name := range flag.Args() {
		if err := processFile(name, delim, *stripSpace); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
