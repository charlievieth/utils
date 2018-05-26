package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Reader struct {
	b   *bufio.Reader
	buf []byte
	out []byte
}

const defaultSize = 32 * 1024

func NewReader(rd io.Reader) *Reader {
	return &Reader{
		b:   bufio.NewReaderSize(rd, defaultSize),
		buf: make([]byte, 0, 128),
		out: make([]byte, 0, 128),
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
	return r.StripANSI(), err
}

func (r *Reader) StripANSI() []byte {
	r.out = r.out[:0]
	for {
		start, end := findIndex(r.buf)
		if start == -1 {
			break
		}
		if start > 0 {
			r.out = append(r.out, r.buf[:start]...)
		}
		if end < len(r.buf) {
			r.buf = r.buf[end:]
		} else {
			r.buf = nil
			break
		}
	}
	r.out = append(r.out, r.buf...)
	return r.out
}

func (r *Reader) ReadAll(wr io.Writer) error {
	out := bufio.NewWriterSize(wr, defaultSize)
	var buf []byte
	var err error
	for {
		buf, err = r.ReadBytes('\n')
		if err != nil {
			break
		}
		if _, err := out.Write(buf); err != nil {
			return errors.New("writing: " + err.Error())
		}
	}
	if err != io.EOF {
		return fmt.Errorf("reading: %s\n", err)
	}
	if _, err := out.Write(buf); err != nil {
		return errors.New("writing: " + err.Error())
	}
	if err := out.Flush(); err != nil {
		return errors.New("flushing: " + err.Error())
	}
	return nil
}

func findIndex(b []byte) (int, int) {
	// Pattern: \x1b\[[0-?]*[ -/]*[@-~]
	const minLen = 2 // "\\[[@-~]"

	start := bytes.IndexByte(b, '\x1b')
	if start == -1 || len(b)-start < minLen || b[start+1]-'@' > '_'-'@' {
		return -1, -1
	}

	n := start + 2 // ESC + second byte [@-_]

	// parameter bytes
	for ; n < len(b) && b[n]-'0' <= '?'-'0'; n++ {
	}
	// intermediate bytes
	for ; n < len(b) && b[n]-' ' <= '/'-' '; n++ {
	}
	// final byte
	if n < len(b) && b[n]-'@' <= '~'-'@' {
		return start, n + 1
	}
	return -1, -1
}

func tempFile(path string) (*os.File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return ioutil.TempFile(filepath.Dir(abs), ".nocolor.")
}

func stripFile(name string) error {
	fi, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("opening file (%s): %s", name, err)
	}
	defer fi.Close()
	if err := NewReader(fi).ReadAll(os.Stdout); err != nil {
		return fmt.Errorf("stripping (%s): %s", name, err)
	}
	return nil
}

func stripFileInPlace(name string) error {
	fi, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("opening file (%s): %s", name, err)
	}
	defer fi.Close()
	fo, err := tempFile(name)
	if err != nil {
		return fmt.Errorf("creating temp file: %s", err)
	}
	tempname := fo.Name()
	exit := func(format string, a ...interface{}) error {
		fo.Close()
		os.Remove(tempname)
		return fmt.Errorf(format, a...)
	}
	if err := NewReader(fi).ReadAll(fo); err != nil {
		return fmt.Errorf("stripping (%s): %s", name, err)
	}
	if err := fi.Close(); err != nil {
		return exit("closing (%s): %s", name, err)
	}
	if err := fo.Close(); err != nil {
		return exit("closing (%s): %s", tempname, err)
	}
	if err := os.Rename(tempname, name); err != nil {
		return exit("rename (%s -> %s): %s", tempname, name, err)
	}
	return nil
}

var InPlace bool

func parseFlags() {
	const usageMsg = "Usage: %[1]s [FILENAME...]\n\n" +
		"  Strips ANSI escape sequences from FILENAME.  If no FILENAME is given\n" +
		"  %[1]s reads from standard input.\n\n"
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usageMsg,
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.BoolVar(&InPlace, "i", false, "edit files in place (short hand)")
	flag.BoolVar(&InPlace, "in-place", false, "edit files in place")
	flag.Parse()
}

func realMain() error {
	parseFlags()
	if flag.NArg() == 0 {
		return NewReader(os.Stdin).ReadAll(os.Stdout)
	}
	var wg sync.WaitGroup
	errs := make(chan error, flag.NArg())
	gate := make(chan struct{}, runtime.NumCPU())
	for _, name := range flag.Args() {
		wg.Add(1)
		go func(name string, wg *sync.WaitGroup, gate chan struct{}) {
			gate <- struct{}{}
			defer func() { <-gate; wg.Done() }()
			var err error
			if InPlace {
				err = stripFileInPlace(name)
			} else {
				err = stripFile(name)
			}
			if err != nil {
				errs <- err
			}
		}(name, &wg, gate)
	}
	wg.Wait()
	close(errs)
	if len(errs) > 0 {
		for e := range errs {
			fmt.Fprintln(os.Stderr, "error:", e)
		}
		return fmt.Errorf("encountered %d errors", len(errs))
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
