package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

func formatJSON(r io.Reader) ([]byte, error) {
	var v interface{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "    ")
}

type FormatFn func(dst *bytes.Buffer, src []byte) error

func formatFile(name string, buf *bytes.Buffer, fn FormatFn) error {
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	buf.Reset()
	buf.Grow(int(fi.Size() + bytes.MinRead))

	if _, err := buf.ReadFrom(f); err != nil {
		return err
	}
	src := make([]byte, buf.Len())
	copy(src, buf.Bytes())
	buf.Reset()
	if err := fn(buf, src); err != nil {
		return err
	}

	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	n, err := buf.WriteTo(f)
	if err != nil {
		return err
	}
	if err := f.Truncate(n); err != nil {
		return err
	}
	return f.Close()
}

func formatFiles(names []string, fn FormatFn) error {
	numCPU := runtime.NumCPU()
	if n := len(names); n < numCPU {
		numCPU = n
	}
	in := make(chan string, numCPU)

	errCount := new(int64)
	var wg sync.WaitGroup
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func(in chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			var buf bytes.Buffer
			for name := range in {
				if err := formatFile(name, &buf, fn); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
					atomic.AddInt64(errCount, 1)
				}
			}
		}(in, &wg)
	}

	for _, name := range names {
		in <- name
	}
	close(in)

	wg.Wait()
	if n := *errCount; n != 0 {
		return fmt.Errorf("encountered %d errors", n)
	}
	return nil
}

var (
	Indent  uint
	UseTabs bool
	Compact bool
)

func parseFlags() {
	const DefaultIndent = 4
	flag.BoolVar(&UseTabs, "tabs", false, "Indent using tabs")
	flag.BoolVar(&UseTabs, "t", false, "Indent using tabs (shorthand)")
	flag.UintVar(&Indent, "indent", DefaultIndent, "Indent this many spaces")
	flag.UintVar(&Indent, "n", DefaultIndent, "Indent this many spaces (shorthand)")
	flag.BoolVar(&Compact, "c", false, "Compact JSON")
	flag.Parse()
}

// (dst *bytes.Buffer, src []byte) error
func main() {
	parseFlags()
	var delim string
	if UseTabs {
		delim = "\t"
	} else {
		delim = strings.Repeat(" ", int(Indent))
	}
	var fn FormatFn
	if Compact {
		fn = json.Compact
	} else {
		fn = func(dst *bytes.Buffer, src []byte) error {
			return json.Indent(dst, src, "", delim)
		}
	}
	if err := formatFiles(flag.Args(), fn); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
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
