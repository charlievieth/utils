package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"text/tabwriter"

	"github.com/charlievieth/pkgs/fastwalk"
)

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

type Walker struct {
	exts map[string]int64
	mu   sync.Mutex
}

func (w *Walker) Walk(path string, typ os.FileMode) error {
	if typ.IsRegular() {
		lines, err := LineCount(path, nil)
		if err != nil {
			return err
		}
		w.mu.Lock()
		w.exts[filepath.Ext(path)] += lines
		w.mu.Unlock()
		return nil
	}
	if typ == os.ModeDir {
		base := filepath.Base(path)
		if base == "" || base[0] == '.' || base[0] == '_' ||
			base == "testdata" || base == "node_modules" {
			return filepath.SkipDir
		}
		return nil
	}
	return nil
}

// CEV: awful name - fixme
type Count struct {
	S, L string
	N    int64
}

type byName []Count

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].L < b[j].L }

type byCount []Count

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].N < b[j].N }

type byNameCount []Count

func (b byNameCount) Len() int      { return len(b) }
func (b byNameCount) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byNameCount) Less(i, j int) bool {
	return b[i].N < b[j].N || (b[i].N == b[j].N && b[i].S < b[j].S)
}

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	args := os.Args[1:]
	if len(args) == 0 {
		args = append(args, pwd)
	}

	w := Walker{
		exts: make(map[string]int64),
	}
	for _, path := range args {
		if !filepath.IsAbs(path) {
			path = filepath.Join(pwd, path)
		}
		if !isDir(path) {
			fmt.Fprintf(os.Stderr, "%s: skipping not a directory\n", path)
			continue
		}
		if err := fastwalk.Walk(path, w.Walk); err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %s\n", path, err)
		}
	}

	var total int64
	exts := make([]Count, 0, len(w.exts))
	for s, n := range w.exts {
		if s == "" {
			s = "<none>"
		}
		exts = append(exts, Count{S: s, N: n})
		total += n
	}

	sort.Sort(byNameCount(exts))

	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, l := range exts {
		b = b[:0]
		b = strconv.AppendInt(b, l.N, 10)
		b = append(b, ':')
		b = append(b, '\t')
		b = append(b, l.S...)
		b = append(b, '\n')
		if _, err := wr.Write(b); err != nil {
			Fatal(err)
		}
	}
	// TODO: print total
	if err := wr.Flush(); err != nil {
		Fatal(err)
	}
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	errMsg := "Error"
	if _, file, line, _ := runtime.Caller(1); file != "" {
		errMsg = fmt.Sprintf("Error (%s:#%d)", filepath.Base(file), line)
	}
	switch e := err.(type) {
	case string, error, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", errMsg, e)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", errMsg, e)
	}
	os.Exit(1)
}

/*
func LineCount(r io.Reader, buf []byte) (lines int64, err error) {
	if buf == nil {
		buf = make([]byte, 32*1024)
	}
	for {
		nr, er := r.Read(buf)
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
*/
