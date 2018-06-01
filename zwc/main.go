package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
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

func LineCount(filename string, buf []byte) (lines int64, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	if buf == nil {
		buf = make([]byte, 8*1024) // Previously 32k
	}
	for {
		nr, er := f.Read(buf)
		if nr > 0 {
			lines += int64(bytes.Count(buf[0:nr], []byte{'\n'}))
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}

func Worker(wg *sync.WaitGroup, count *int64, filenames <-chan string, errCh chan<- error) {
	defer wg.Done()
	buf := make([]byte, 8*1024)
	for name := range filenames {
		n, err := LineCount(name, buf)
		if err != nil {
			errCh <- err
			continue
		}
		atomic.AddInt64(count, n)
	}
}

func ReadInput(r *Reader, delim byte, filenames chan<- string, errCh chan<- error) {
	defer close(filenames)
	var buf []byte
	var err error
	for {
		buf, err = r.ReadBytes(delim)
		if err != nil {
			break
		}
		if len(buf) > 1 {
			filenames <- string(buf[:len(buf)-1])
		}
	}
	if err != io.EOF {
		errCh <- err
		return
	}
	if len(buf) > 1 {
		filenames <- string(buf[:len(buf)-1])
	}
}

func main() {
	var wg sync.WaitGroup
	var count int64
	filenames := make(chan string, 64)
	errCh := make(chan error, 64)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go Worker(&wg, &count, filenames, errCh)
	}

	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 128),
	}
	go ReadInput(&r, 0, filenames, errCh)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

Loop:
	for {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "error: %s", err)
			continue Loop
		case <-done:
			break Loop
		}
	}
	fmt.Println("Count:", count)
}
