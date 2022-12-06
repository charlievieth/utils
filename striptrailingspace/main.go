package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("[strip] ")
}

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		b:   bufio.NewReader(r),
		buf: make([]byte, 0, 128),
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

func (r *Reader) Reset(rd io.Reader) {
	r.b.Reset(rd)
	r.buf = r.buf[:0]
}

var readerPool = sync.Pool{
	New: func() any {
		return NewReader(nil)
	},
}

func readerPoolGet(rd io.Reader) *Reader {
	r := readerPool.Get().(*Reader)
	r.Reset(rd)
	return r
}

func readerPoolPut(r *Reader) {
	r.Reset(nil)
	readerPool.Put(r)
}

var writerPool = sync.Pool{
	New: func() any {
		return bufio.NewWriterSize(nil, 32*1024)
	},
}

func writerPoolGet(wr io.Writer) *bufio.Writer {
	w := writerPool.Get().(*bufio.Writer)
	w.Reset(wr)
	return w
}

func writerPoolPut(w *bufio.Writer) {
	w.Reset(nil)
	writerPool.Put(w)
}

func isSpace(r rune) bool {
	return r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r' ||
		r == ' ' || r == 0x85 || r == 0xA0
}

func Strip(rd io.Reader, wr io.Writer, trailingNewline bool) error {
	r := readerPoolGet(rd)
	defer readerPoolPut(r)

	w := writerPoolGet(wr)
	defer writerPoolPut(w)

	var err error
	var suffix string
	for {
		b, e := r.ReadBytes('\n')
		if n := len(b); n != 0 {
			switch {
			case n >= 2 && b[n-2] == '\r' && b[n-1] == '\n':
				suffix = "\r\n"
			case n >= 1 && b[n-1] == '\n':
				suffix = "\n"
			default:
				// No newline at end of file
				if trailingNewline {
					if suffix == "" {
						suffix = "\n"
					}
				} else {
					suffix = ""
				}
			}
			b = append(bytes.TrimRightFunc(b, isSpace), suffix...)
			if _, ew := w.Write(b); ew != nil && e == nil {
				e = ew
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if err != nil {
		return err
	}
	return w.Flush()
}

func StripFile(path string, trailingNewline bool) error {
	dir, name := filepath.Split(path)
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fi.Close()

	tmp, err := os.CreateTemp(dir, name+".strip.*")
	if err != nil {
		return err
	}
	tmpname := tmp.Name()

	err = Strip(fi, tmp, trailingNewline)
	fi.Close()
	if cerr := tmp.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmpname)
		return err
	}

	if err := os.Rename(tmpname, path); err != nil {
		os.Remove(tmpname)
		return err
	}
	return nil
}

func StripFiles(paths []string) error {
	numWorkers := runtime.NumCPU()
	if len(paths) < numWorkers {
		numWorkers = len(paths)
	}

	ch := make(chan string, numWorkers)
	var wg sync.WaitGroup
	var fails int32
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range ch {
				if err := StripFile(path, false); err != nil {
					log.Printf("%s: %v\n", path, err)
					atomic.AddInt32(&fails, 1)
				}
			}
		}()
	}

	for _, path := range paths {
		ch <- path
	}
	close(ch)
	wg.Wait()

	if n := atomic.LoadInt32(&fails); n > 0 {
		return fmt.Errorf("found %d errors stripping files", n)
	}
	return nil
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "USAGE: FILENAME...")
		os.Exit(1)
	}
	if err := StripFiles(os.Args[1:]); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
