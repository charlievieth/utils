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
)

func formatJSON(r io.Reader) ([]byte, error) {
	var v interface{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "    ")
}

func formatFile(name, delim string, buf *bytes.Buffer) error {
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
	if err := json.Indent(buf, src, "", delim); err != nil {
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

func formatFiles(names []string, delim string) {
	numCPU := runtime.NumCPU()
	if n := len(names); n < numCPU {
		numCPU = n
	}
	in := make(chan string, numCPU)

	var wg sync.WaitGroup
	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func(in chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			var buf bytes.Buffer
			for name := range in {
				if err := formatFile(name, delim, &buf); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %s\n", name, err)
				}
			}
		}(in, &wg)
	}

	for _, name := range names {
		in <- name
	}
	close(in)

	wg.Wait()
}

var (
	Indent  uint
	UseTabs bool
)

func parseFlags() {
	const DefaultIndent = 4
	flag.BoolVar(&UseTabs, "tabs", false, "Indent using tabs")
	flag.BoolVar(&UseTabs, "t", false, "Indent using tabs (shorthand)")
	flag.UintVar(&Indent, "indent", DefaultIndent, "Indent this many spaces")
	flag.UintVar(&Indent, "n", DefaultIndent, "Indent this many spaces (shorthand)")
	flag.Parse()
}

func main() {
	parseFlags()
	var delim string
	if UseTabs {
		delim = "\t"
	} else {
		delim = strings.Repeat(" ", int(Indent))
	}
	formatFiles(flag.Args(), delim)
	return

	var v interface{}
	if err := json.NewDecoder(os.Stdin).Decode(v); err != nil {
		Fatal(err)
	}
	b, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		Fatal(err)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		Fatal(err)
	}
	// json.MarshalIndent(v, prefix, indent)
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
