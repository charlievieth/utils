package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/charlievieth/utils/pathutils"
)

var ZeroDelim bool
var ZeroTerm bool

func parseFlags() {
	flag.BoolVar(&ZeroDelim, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	flag.BoolVar(&ZeroTerm, "z", false,
		"End each output line with NUL ('\\0'), not newline")
	flag.Parse()
}

func main() {
	parseFlags()
	delim := byte('\n')
	if ZeroDelim {
		delim = 0
	}
	eol := byte('\n')
	if ZeroTerm {
		eol = 0
	}
	r := pathutils.NewReader(bufio.NewReader(os.Stdin))
	out := bufio.NewWriter(os.Stdout)
	var err error
	for {
		b, e := r.ReadBytes(delim)
		if len(b) != 0 {
			if b = pathutils.Dir(b); len(b) != 0 {
				if _, ew := out.Write(append(b, eol)); ew != nil {
					if e == nil {
						e = ew
					}
				}
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if e := out.Flush(); e != nil && err == nil {
		err = e
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
