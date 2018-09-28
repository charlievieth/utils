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

var NullTerminate bool

func parseFlags() {
	const usageMsg = "Usage: %[1]s [FILENAME]\n\n" +
		"  Prints the file extension of FILENAME.  If no FILENAME is given\n" +
		"  %[1]s reads from standard input.\n\n"

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usageMsg,
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.BoolVar(&NullTerminate, "0", false,
		"Expect NUL ('\\0') characters as separators, instead of newlines\n"+
			"when reading from standard input")
	flag.Parse()
}

func readStdin() error {
	delim := byte('\n')
	if NullTerminate {
		delim = 0
	}
	r := pathutils.NewReader(bufio.NewReader(os.Stdin))
	out := bufio.NewWriter(os.Stdout)
	var err error
	for {
		b, e := r.ReadBytes(delim)
		b = pathutils.Ext(b)
		if len(b) != 0 {
			if _, ew := out.Write(append(b, '\n')); ew != nil {
				if e == nil {
					e = ew
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
	if err != nil {
		return err
	}
	return out.Flush()
}

func realMain() error {
	parseFlags()
	if flag.NArg() == 0 || flag.Arg(0) == "-" {
		return readStdin()
	}
	for _, s := range flag.Args() {
		fmt.Println(filepath.Ext(s))
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return
}
