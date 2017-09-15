package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"text/tabwriter"
	"unicode/utf8"
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

var NullTerminate bool

func parseFlags() {
	flag.BoolVar(&NullTerminate, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	flag.Parse()
}

type Response struct {
	Name  string
	Count int
}

type byName []Response

func (b byName) Len() int           { return len(b) }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type byCount []Response

func (b byCount) Len() int           { return len(b) }
func (b byCount) Less(i, j int) bool { return b[i].Count < b[j].Count }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func worker(fileCh <-chan string, resCh chan<- []Response) {
	var rs []Response
	for filename := range fileCh {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			continue
		}
		n := 0
		for _, c := range b {
			if c&utf8.RuneSelf != 0 {
				n++
			}
		}
		rs = append(rs, Response{
			Name:  filename,
			Count: n,
		})
	}
	resCh <- rs
}

func realMain() error {
	parseFlags()
	r := Reader{
		b:   bufio.NewReaderSize(os.Stdin, 4096),
		buf: make([]byte, 0, 128),
	}
	var (
		buf   []byte
		err   error
		delim byte
	)
	if !NullTerminate {
		delim = '\n'
	}

	numCPU := runtime.NumCPU()
	if numCPU > 10 {
		numCPU = 10
	}
	fileCh := make(chan string)
	resCh := make([]chan []Response, numCPU)

	for i := 0; i < numCPU; i++ {
		resCh[i] = make(chan []Response)
		go worker(fileCh, resCh[i])
	}

	for {
		buf, err = r.ReadBytes(delim)
		if err != nil {
			break
		}
		if buf[len(buf)-1] == delim {
			buf = buf[:len(buf)-1]
		}
		fileCh <- string(buf)
	}
	close(fileCh)

	var res []Response
	for i := 0; i < numCPU; i++ {
		r := <-resCh[i]
		res = append(res, r...)
	}

	sort.Sort(byCount(res))
	sort.Stable(byName(res))

	NotAscii := 0

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, r := range res {
		if r.Count < 1 {
			continue
		}
		NotAscii++
		b = b[:0]
		b = append(b, r.Name...)
		b = append(b, ':')
		b = append(b, '\t')
		b = strconv.AppendInt(b, int64(r.Count), 10)
		b = append(b, '\n')
		if _, err := w.Write(b); err != nil {
			Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		Fatal(err)
	}
	fmt.Print("\n")
	fmt.Printf("Total:   %d\n", len(res))
	fmt.Printf("ASCII:   %d\n", len(res)-NotAscii)
	fmt.Printf("UNICODE: %d\n", NotAscii)

	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return
}

func Fatal(err interface{}) {
	if err != nil {
		var s string
		if _, file, line, ok := runtime.Caller(1); ok && file != "" {
			s = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
