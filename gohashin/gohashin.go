package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"text/tabwriter"

	"github.com/spf13/pflag"
)

type hashWriter struct {
	h    hash.Hash
	name string
}

// TODO: we can make this so that we don't need for it to be atomic
type atomicWriter struct {
	w   io.Writer
	err atomic.Value
}

type errorValue struct {
	Err error
}

func (w *atomicWriter) write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	return n, nil
}

func (w *atomicWriter) Write(p []byte) (int, error) {
	if err := w.Error(); err != nil {
		return 0, err
	}
	n, err := w.write(p)
	if err != nil {
		w.err.Store(&errorValue{err})
	}
	return n, err
}

func (w *atomicWriter) Error() error {
	if v := w.err.Load(); v != nil {
		return v.(*errorValue).Err
	}
	return nil
}

type multiWriter struct {
	writers []*atomicWriter
	ch      []chan []byte
	wg      sync.WaitGroup
	closed  bool
}

func NewMultiWriter(writers ...io.Writer) *multiWriter {
	aws := make([]*atomicWriter, len(writers))
	for i := range writers {
		aws[i] = &atomicWriter{w: writers[i]}
	}
	return &multiWriter{writers: aws}
}

func (t *multiWriter) lazyInit() {
	if t.ch != nil {
		return
	}
	n := len(t.writers)
	t.ch = make([]chan []byte, n)
	t.wg.Add(n)
	for i := range t.ch {
		t.ch[i] = make(chan []byte, 1)
		go func(w *atomicWriter, ch chan []byte) {
			defer t.wg.Done()
			for p := range ch {
				_, _ = w.Write(p)
			}
		}(t.writers[i], t.ch[i])
	}
}

func (t *multiWriter) Error() error {
	for _, w := range t.writers {
		if err := w.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (t *multiWriter) Write(p []byte) (int, error) {
	if err := t.Error(); err != nil {
		return 0, err
	}
	if t.ch == nil {
		t.lazyInit()
	}
	b := make([]byte, len(p))
	copy(b, p)
	for _, ch := range t.ch {
		ch <- b
	}
	return len(p), nil
}

func (t *multiWriter) Close() error {
	if t.ch == nil {
		return errors.New("multiWriter: never initialized")
	}
	if t.closed {
		return errors.New("multiWriter: already closed")
	}
	t.closed = true
	for i := range t.ch {
		close(t.ch[i])
	}
	t.wg.Wait()
	return t.Error()
}

func main() {
	printSize := pflag.Bool("size", false, "print size")
	hashMD5 := pflag.Bool("md5", false, "calculate md5")
	hashSHA1 := pflag.Bool("sha1", false, "calculate sha1")
	hashSHA224 := pflag.Bool("sha224", false, "calculate sha224")
	hashSHA256 := pflag.Bool("sha256", false, "calculate sha256")
	hashSHA384 := pflag.Bool("sha384", false, "calculate SHA384")
	hashSHA512_224 := pflag.Bool("sha512_224", false, "calculate SHA512_224")
	hashSHA512_256 := pflag.Bool("sha512_256", false, "calculate SHA512_256")
	hashSHA512 := pflag.Bool("sha512", false, "calculate sha512")
	pflag.Parse()

	// default to MD5
	if !*hashMD5 && !*hashSHA1 && !*hashSHA224 && !*hashSHA256 &&
		!*hashSHA384 && !*hashSHA512_224 && !*hashSHA512_256 && !*hashSHA512 {
		*hashMD5 = true
	}

	var hashes []hashWriter
	if *hashMD5 {
		hashes = append(hashes, hashWriter{md5.New(), "md5"})
	}
	if *hashSHA1 {
		hashes = append(hashes, hashWriter{sha1.New(), "sha1"})
	}
	if *hashSHA224 {
		hashes = append(hashes, hashWriter{sha256.New224(), "sha224"})
	}
	if *hashSHA256 {
		hashes = append(hashes, hashWriter{sha256.New(), "sha256"})
	}
	if *hashSHA384 {
		hashes = append(hashes, hashWriter{sha512.New384(), "sha384"})
	}
	if *hashSHA512_224 {
		hashes = append(hashes, hashWriter{sha512.New512_224(), "sha512_224"})
	}
	if *hashSHA512_256 {
		hashes = append(hashes, hashWriter{sha512.New512_256(), "sha512_256"})
	}
	if *hashSHA512 {
		hashes = append(hashes, hashWriter{sha512.New(), "sha512"})
	}

	writers := []io.Writer{os.Stdout}
	for _, h := range hashes {
		writers = append(writers, h.h)
	}
	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i].name < hashes[j].name
	})

	buf := make([]byte, 64*1024)
	w := NewMultiWriter(writers...)
	size, err := io.CopyBuffer(w, os.Stdin, buf)
	if err != nil {
		Fatal(err)
	}
	if err := w.Close(); err != nil {
		Fatal(err)
	}

	tw := tabwriter.NewWriter(os.Stderr, 2, 0, 2, ' ', 0)
	for _, h := range hashes {
		_, err := fmt.Fprintf(tw, "%s:\t%x\n", h.name, h.h.Sum(nil))
		if err != nil {
			Fatal(err)
		}
	}
	if *printSize {
		_, err := fmt.Fprintf(tw, "%s:\t%d\n", "size", size)
		if err != nil {
			Fatal(err)
		}
	}
	if err := tw.Flush(); err != nil {
		Fatal(err)
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
