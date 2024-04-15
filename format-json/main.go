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

func formatFile(name, indent string, sortKeys bool, buf *bytes.Buffer) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	dir, base := filepath.Split(name)
	tmp, err := os.CreateTemp(dir, base+".*")
	if err != nil {
		return err
	}
	exit := func(err error) error {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}

	buf.Reset()
	dec := json.NewDecoder(f)
	for {
		if sortKeys {
			var v any
			if err = dec.Decode(&v); err != nil {
				if err == io.EOF {
					err = nil
				}
				break // ok
			}
			data, err := json.MarshalIndent(v, "", indent)
			if err != nil {
				break
			}
			buf.Reset()
			buf.Write(data) // TODO: semi-useless copy
		} else {
			var m json.RawMessage
			if err = dec.Decode(&m); err != nil {
				if err == io.EOF {
					err = nil
				}
				break // ok
			}
			buf.Reset()
			if err = json.Indent(buf, m, "", indent); err != nil {
				break
			}
		}
		buf.WriteByte('\n')
		if _, err = buf.WriteTo(tmp); err != nil {
			break
		}
	}
	if err != nil {
		return exit(err)
	}

	if err := f.Close(); err != nil {
		return exit(err)
	}
	if err := tmp.Chmod(fi.Mode()); err != nil {
		return exit(err)
	}
	if err := tmp.Close(); err != nil {
		return exit(err)
	}
	if err := os.Rename(tmp.Name(), name); err != nil {
		return exit(err)
	}
	return nil
}

func formatFiles(names []string, indent string, sortKeys bool) error {
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
				if err := formatFile(name, indent, sortKeys, &buf); err != nil {
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
	Indent   uint
	UseTabs  bool
	SortKeys bool
	Compact  bool
)

func parseFlags() {
	const DefaultIndent = 4
	flag.BoolVar(&UseTabs, "tabs", false, "Indent using tabs")
	flag.BoolVar(&UseTabs, "t", false, "Indent using tabs (shorthand)")
	flag.BoolVar(&SortKeys, "sort", false, "Sort keys")
	flag.BoolVar(&SortKeys, "s", false, "Sort keys (shorthand)")
	flag.UintVar(&Indent, "indent", DefaultIndent, "Indent this many spaces")
	flag.UintVar(&Indent, "n", DefaultIndent, "Indent this many spaces (shorthand)")
	flag.BoolVar(&Compact, "c", false, "Compact JSON")
	flag.Parse()
}

func main() {
	parseFlags()
	var delim string
	switch {
	case UseTabs:
		delim = "\t"
	case Compact:
		delim = ""
	default:
		delim = strings.Repeat(" ", int(Indent))
	}
	if err := formatFiles(flag.Args(), delim, SortKeys); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
