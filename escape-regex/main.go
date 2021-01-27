package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	flag "github.com/spf13/pflag"
)

type Mode int

const (
	EscapeSlash Mode = 1 << iota
	EscapeAll
)

func QuoteMeta(s string, mode Mode) string {
	if mode&EscapeAll == 0 {
		s = regexp.QuoteMeta(s)
		if mode&EscapeSlash != 0 {
			// WARN: this is will break if '/' is already escaped
			s = strings.Replace(s, `/`, `\/`, -1)
		}
		return s
	}
	var b strings.Builder
	n := utf8.RuneCountInString(s)
	b.Grow(n * 3)
	for _, r := range s {
		b.WriteByte('[')
		b.WriteRune(r)
		b.WriteByte(']')
	}
	return b.String()
}

func ReadStdin(mode Mode) (err error) {
	r := bufio.NewReader(os.Stdin)
	for {
		b, e := r.ReadBytes('\n')
		if len(b) != 0 {
			fmt.Println(QuoteMeta(string(b), mode))
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	return err
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s: [OPTION]... [PATTERN]...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	slash := flag.BoolP("slash", "s", false, "Escape forward slashes")
	all := flag.BoolP("all", "a", false, "Safely escape all characters.")
	flag.Parse()

	var mode Mode
	if *slash {
		mode |= EscapeSlash
	}
	if *all {
		mode |= EscapeAll
	}

	if flag.NArg() == 0 || flag.NArg() == 1 && flag.Arg(0) == "-" {
		if err := ReadStdin(mode); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	} else {
		for _, s := range flag.Args() {
			fmt.Println(QuoteMeta(s, mode))
		}
	}
}
