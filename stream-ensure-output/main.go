package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	escapeRe *regexp.Regexp
	initRe   sync.Once
)

func Unescape(b []byte) []byte {
	s := string(bytes.TrimPrefix(b, []byte("\\x")))
	if n, err := strconv.ParseInt(s, 16, 8); err == nil {
		return []byte{byte(n)}
	}
	return b
}

func ReplaceControlChars(src []byte, escapeSingleNewline bool) []byte {
	initRe.Do(func() {
		escapeRe = regexp.MustCompile(`\\x[[:xdigit:]]{2}`)
	})
	b := bytes.ReplaceAll(src, []byte(`\r\n`), []byte{'\n'})
	// WARN
	if escapeSingleNewline {
		b = bytes.ReplaceAll(src, []byte(`\n`), []byte{'\n'})
	}
	b = bytes.ReplaceAll(b, []byte(`\t`), []byte{'\t'})
	return escapeRe.ReplaceAllFunc(b, Unescape)
}

func Stream(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	var err error
	for {
		b, er := br.ReadBytes('\n')
		if len(b) != 0 {
			_, ew := w.Write(append(ReplaceControlChars(b, false), '\n'))
			if ew != nil {
				if er == nil || er == io.EOF {
					er = ew
				}
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
}

func ReplaceFile(name string) error {
	src, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}
	singleRe := regexp.MustCompile(`(?m)b'[^']+'`)
	src = singleRe.ReplaceAllFunc(src, func(b []byte) []byte {
		b = bytes.TrimPrefix(b, []byte(`b'`))
		b = bytes.TrimSuffix(b, []byte{'\''})
		if len(b) == 0 {
			return b
		}
		return ReplaceControlChars(b, true)
	})
	doubleRe := regexp.MustCompile(`(?m)b"[^"]+"`)
	src = doubleRe.ReplaceAllFunc(src, func(b []byte) []byte {
		b = bytes.TrimPrefix(b, []byte(`b"`))
		b = bytes.TrimSuffix(b, []byte{'"'})
		if len(b) == 0 {
			return b
		}
		return ReplaceControlChars(b, true)
	})
	if _, err := os.Stdout.Write(src); err != nil {
		return err
	}
	return nil
}

func IndentJSON(src []byte) []byte {
	// re := regexp.
	return nil
}

func Usage() {
	const msg = "Usage: %s [STRING]...\n" +
		"Unescape Python binary data (by default stdin is read from)\n"
	fmt.Fprintf(os.Stderr, msg, filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

func realMain() error {
	// WARN: this is just a hack as it kinda breaks the current design
	var filename string
	flag.StringVar(&filename, "f", "", "Filename to read from")
	flag.Usage = Usage
	flag.Parse()

	// WARN: this kinda breaks the design
	if filename != "" {
		return ReplaceFile(filename)
	}

	if n := flag.NArg(); n == 0 || (n == 1 && flag.Arg(0) == "-") {
		if err := Stream(os.Stdin, os.Stdout); err != nil {
			return err
		}
		return nil
	}
	for _, s := range flag.Args() {
		r := strings.NewReader(s)
		if err := Stream(r, os.Stdout); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
