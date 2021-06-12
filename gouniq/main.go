package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/charlievieth/utils/pathutils"
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
		b:   bufio.NewReaderSize(in, 8192),
		buf: make([]byte, 128),
	}
	w := bufio.NewWriterSize(out, 8192)
	seen := make(map[string]struct{})
	var err error
	for {
		b, er := r.ReadBytes(delim)
		b = trimSpace(b)
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

// type Set struct {
// 	m map[string]struct{}
// }

// func (s *Set) Add(b []byte) bool {
// 	_, found := s.m[string(b)]
// 	if !found {
// 		s.m[string(b)] = struct{}{}
// 	}
// 	return !found
// }

func processStdin(trim, lineBuffer bool, delim byte) error {
	var (
		bw *bufio.Writer
		w  io.Writer = os.Stdout
	)
	if !lineBuffer {
		bw = bufio.NewWriter(os.Stdout)
		w = bw
	}

	r := pathutils.NewReader(bufio.NewReaderSize(os.Stdin, 8192))
	seen := make(map[string]struct{}, 128)

	var err error
	for {
		b, er := r.ReadBytes(delim)
		if trim {
			b = trimSpace(b)
		}
		if len(b) != 0 {
			if _, ok := seen[string(b)]; !ok {
				seen[string(b)] = struct{}{}
				if _, ew := w.Write(append(b, '\n')); ew != nil {
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
	if bw != nil {
		bw.Flush()
	}
	return err
}

type Line struct {
	N    int
	Data []byte
}

// CEV: this is a bad/hard idea
/*
func processStdinParallel(trim, lineBuffer bool, delim byte) error {
	var (
		bw *bufio.Writer
		w  io.Writer = os.Stdout
	)
	if !lineBuffer {
		bw = bufio.NewWriter(os.Stdout)
		w = bw
	}

	var lines []*Line
	numCPU := runtime.NumCPU()
	if numCPU > 4 {
		numCPU--
	}
	ch := make(chan *Line, numCPU*8)
	for i := 0; i < numCPU; i++ {
		go func() {
			for ll := range ch {

			}
		}()
	}

	r := pathutils.NewReader(bufio.NewReaderSize(os.Stdin, 8192))
	seen := make(map[string]struct{}, 128)

	var err error
	for {
		b, er := r.ReadBytes(delim)
		if trim {
			b = trimSpace(b)
		}
		if len(b) != 0 {
			if _, ok := seen[string(b)]; !ok {
				seen[string(b)] = struct{}{}
				if _, ew := w.Write(append(b, '\n')); ew != nil {
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
	if bw != nil {
		bw.Flush()
	}
	return err
}
*/

func xmain() {
	// flag.Usage = func() {
	// 	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	// 	flag.PrintDefaults()
	// }
	trimspace := flag.Bool("s", false, "Trim leading and trailing whitespace form lines.")
	zeroDelim := flag.Bool("0", false, "Lines are 0/NULL terminated.")
	isTerm := term.IsTerminal(2)

	_ = trimspace
	_ = zeroDelim
	_ = isTerm
}
