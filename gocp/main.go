package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charlievieth/utils/gocp/fastwalk"
)

var seenDirs sync.Map

func MkdirAll(path string) error {
	if _, ok := seenDirs.Load(path); ok {
		return nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if _, loaded := seenDirs.LoadOrStore(path, struct{}{}); loaded {
		fmt.Fprintf(os.Stderr, "MkdirAll duplicate: %s\n", path)
	}
	return nil
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func CopyFile(src, dst string, fi os.FileInfo) error {
	if err := MkdirAll(filepath.Dir(dst)); err != nil {
		return err
	}

	r, err := os.Open(src)
	if err != nil {
		return err
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(int(fi.Size() + bytes.MinRead))

	_, err = buf.ReadFrom(r)
	r.Close()
	if err != nil {
		bufPool.Put(buf)
		return nil
	}

	w, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fi.Mode())
	if err != nil {
		bufPool.Put(buf)
		return err
	}
	_, err = buf.WriteTo(w)
	bufPool.Put(buf)
	if err != nil {
		return err
	}
	return w.Close()
}

type Walker struct {
	Root        string // root to walk
	Prefix      string // remove this
	Destination string
	seen        int64
	copied      int64
	errors      int64
}

func NewWalker(src, dst string) (*Walker, error) {
	var err error
	src, err = filepath.Abs(src)
	if err != nil {
		return nil, err
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return nil, err
	}
	w := &Walker{
		Root:        src,
		Prefix:      filepath.Dir(src) + "/",
		Destination: dst,
	}
	return w, nil
}

func (w *Walker) Copied() int64 { return atomic.LoadInt64(&w.copied) }
func (w *Walker) Errors() int64 { return atomic.LoadInt64(&w.errors) }
func (w *Walker) Seen() int64   { return atomic.LoadInt64(&w.seen) }

func (w *Walker) Walk(path string, fi os.FileInfo) error {
	atomic.AddInt64(&w.seen, 1)
	if !fi.Mode().IsRegular() {
		return nil
	}
	dst := filepath.Join(w.Destination, strings.TrimPrefix(path, w.Prefix))
	if err := CopyFile(path, dst, fi); err != nil {
		atomic.AddInt64(&w.errors, 1)
		return err
	}
	atomic.AddInt64(&w.copied, 1)
	return nil
}

func ErrorFn(err error) {
	fmt.Fprintln(os.Stderr, err)
}

func main() {
	var numWorkers int
	flag.IntVar(&numWorkers, "n", -1, "Number of workers")
	flag.Parse()
	if flag.NArg() != 2 {
		Fatal("USAGE: [SOURCE] [DESTINATION]")
	}
	fmt.Fprintln(os.Stderr, "Workers:", numWorkers)

	w, err := NewWalker(flag.Arg(0), flag.Arg(2))
	if err != nil {
		Fatal(err)
	}
	t := time.Now()
	if err := fastwalk.Walk(w.Root, w.Walk, ErrorFn, numWorkers); err != nil {
		Fatal(err)
	}
	d := time.Since(t)
	fmt.Println("Copied:", w.Copied())
	fmt.Println("Errors:", w.Errors())
	fmt.Println("Seen:", w.Seen())
	copied := w.Copied()
	if copied == 0 {
		copied = 1
	}
	fmt.Println("Time:", d, d/time.Duration(copied))
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var s string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		s = "Error"
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
	}
	os.Exit(1)
}
