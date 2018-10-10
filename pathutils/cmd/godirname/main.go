package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charlievieth/utils/pathutils"
)

const ProgramName = "godirname"

var ZeroDelim bool
var ZeroTerm bool

func parseFlags() *flag.FlagSet {
	set := flag.NewFlagSet(ProgramName, flag.ExitOnError)

	set.BoolVar(&ZeroDelim, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines")
	set.BoolVar(&ZeroTerm, "z", false,
		"End each output line with NUL ('\\0'), not newline")

	set.Usage = func() {
		fmt.Fprintf(set.Output(), "%s: [OPTIONS] [PATH...]\n", set.Name())
		flag.PrintDefaults()
	}
	// error handled by flag.ExitOnError
	set.Parse(os.Args[1:])
	return set
}

func processStdin(delim, eol byte, out *bufio.Writer) error {
	r := pathutils.NewReader(bufio.NewReader(os.Stdin))
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
	return err
}

func processArgs(eol byte, args []string, out *bufio.Writer) error {
	// special case
	if len(args) == 1 && args[0] == "." {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args[0] = pwd
	}
	for _, s := range args {
		if _, err := out.WriteString(filepath.Dir(s)); err != nil {
			return err
		}
		if err := out.WriteByte(eol); err != nil {
			return err
		}
	}
	return nil
}

func realMain() error {
	set := parseFlags()
	delim := byte('\n')
	if ZeroDelim {
		delim = 0
	}
	eol := byte('\n')
	if ZeroTerm {
		eol = 0
	}
	out := bufio.NewWriter(os.Stdout)
	var err error
	if set.NArg() == 0 || (set.NArg() == 1 && set.Arg(0) == "-") {
		err = processStdin(delim, eol, out)
	} else {
		err = processArgs(eol, set.Args(), out)
	}
	if err != nil {
		return err
	}
	if err := out.Flush(); err == nil {
		return err
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
