package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
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

// Note: includes delim in returned slice
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
	// if n := len(frag); n != 0 && frag[n-1] == delim {
	// 	frag = frag[:n-1]
	// }
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

func isSpace(r byte) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
}

func trimSpaceRight(s []byte) []byte {
	i := len(s) - 1
	for ; i >= 0 && isSpace(s[i]); i-- {
	}
	return s[:i+1]
}

func contextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func processFile(ctx context.Context, name string) error {
	if contextCancelled(ctx) {
		return nil
	}

	fi, err := os.Open(name)
	if err != nil {
		return err
	}
	defer fi.Close()

	dir, base := filepath.Split(name)
	fo, err := os.CreateTemp(dir, base+".*")
	if err != nil {
		return err
	}

	exit := func(err error) error {
		fi.Close()
		fo.Close()
		os.Remove(fo.Name())
		return err
	}

	r := Reader{
		b:   bufio.NewReader(fi),
		buf: make([]byte, 128),
	}
	w := bufio.NewWriter(fo)

	for i := 0; ; i++ {
		var b []byte
		b, err = r.ReadBytes('\n')
		if len(b) != 0 {
			b = append(trimSpaceRight(b), '\n')
			if _, ew := w.Write(b); ew != nil {
				if err == nil || err == io.EOF {
					err = ew
				}
			}
		}
		if err != nil {
			break
		}
		// check if the context is cancelled
		if i%32*1024 == 0 && contextCancelled(ctx) {
			err = ctx.Err()
			break
		}
	}
	if err != io.EOF {
		return exit(err)
	}
	if err := w.Flush(); err != nil {
		return exit(err)
	}
	mode, err := fi.Stat()
	if err != nil {
		return exit(err)
	}
	if err := fo.Chmod(mode.Mode()); err != nil {
		return exit(err)
	}
	if err := fo.Close(); err != nil {
		return exit(err)
	}
	if err := os.Rename(fo.Name(), fi.Name()); err != nil {
		return exit(err)
	}

	return nil
}

func realMain(ctx context.Context) error {
	flag.Parse()
	if flag.NArg() == 0 {
		return errors.New("no arguments")
	}

	var failed int32
	var wg sync.WaitGroup
	gate := make(chan struct{}, runtime.NumCPU()*2)

	for i := 0; i < flag.NArg(); i++ {
		wg.Add(1)
		go func(name string) {
			gate <- struct{}{}
			defer func() {
				wg.Done()
				<-gate
				if err := processFile(ctx, name); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
					atomic.AddInt32(&failed, 1)
				}
			}()
		}(flag.Arg(i))
	}
	wg.Wait()
	if n := atomic.LoadInt32(&failed); n != 0 {
		return fmt.Errorf("failed processing %d files", n)
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := realMain(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
