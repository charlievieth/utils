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

func formatFile(name, indent string, buf *bytes.Buffer) error {
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	buf.Reset()
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	dec := json.NewDecoder(f)
	for {
		var m interface{}
		de := dec.Decode(&m)
		if de != nil && de != io.EOF {
			return de
		}
		if ee := enc.Encode(m); ee != nil {
			if de == nil || de == io.EOF {
				return ee
			}
		}
		if de != nil {
			break // io.EOF
		}
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

func formatFiles(names []string, indent string) error {
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
				if err := formatFile(name, indent, &buf); err != nil {
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
	if err := formatFiles(flag.Args(), delim); err != nil {
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
