package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
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

func ReplaceControlChars(src []byte) []byte {
	initRe.Do(func() {
		escapeRe = regexp.MustCompile(`\\x[[:xdigit:]]{2}`)
	})
	b := bytes.ReplaceAll(src, []byte(`\r\n`), []byte{'\n'})
	b = bytes.ReplaceAll(b, []byte(`\t`), []byte{'\t'})
	return escapeRe.ReplaceAllFunc(b, Unescape)
}

func Stream(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	var err error
	for {
		b, er := br.ReadBytes('\n')
		if len(b) != 0 {
			_, ew := w.Write(append(ReplaceControlChars(b), '\n'))
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

func Usage() {
	const msg = "Usage: %s [STRING]...\n" +
		"Unescape Python binary data (by default stdin is read from)\n"
	fmt.Fprintf(os.Stderr, msg, filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

func realMain() error {
	flag.Usage = Usage
	flag.Parse()
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
