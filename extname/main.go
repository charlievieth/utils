package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func (r *Reader) ReadBytes(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	r.buf = r.buf[:0]
	for {
		var e error
		frag, e = r.b.ReadSlice(delim)
		if e == nil { // got final fragment
			break
		}
		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}
		r.buf = append(r.buf, frag...)
	}
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

func Ext(path []byte) []byte {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return nil
}

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
	r := Reader{
		b:   bufio.NewReader(os.Stdin),
		buf: make([]byte, 0, 128),
	}
	var buf []byte
	var err error
	delim := byte('\n')
	if NullTerminate {
		delim = 0
	}
	for {
		buf, err = r.ReadBytes(delim)
		if err != nil {
			break
		}
		buf = Ext(buf)
		if n := len(buf); n != 0 {
			buf[n-1] = '\n'
		} else {
			buf = append(buf, '\n')
		}
		if _, err := os.Stdout.Write(buf); err != nil {
			return fmt.Errorf("writing: %s\n", err)
		}
	}
	if err != io.EOF {
		return fmt.Errorf("reading: %s\n", err)
	}
	if n := len(buf); n != 0 {
		if buf[n-1] != '\n' {
			buf = append(buf, '\n')
		}
		if _, err := os.Stdout.Write(Ext(buf)); err != nil {
			return fmt.Errorf("writing: %s\n", err)
		}
	}
	return nil
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
