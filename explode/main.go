package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const UsageMsg = `Usage: %[1]s [OPTIONS...] ARGS...
%[1]s explodes strings by character
`

func dropNewline(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		c := b[i]
		if c != '\n' && c != '\r' {
			return b[:i+1]
		}
	}
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), UsageMsg,
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	splitSep := flag.String("split", "", "Separate to split strings by")
	joinSep := flag.String("join", "", "Join exploded string with sep")
	noNewline := flag.Bool("no-newline", false, "Omit newline between exploded words")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		r := bufio.NewReader(os.Stdin)
		var err error
		for {
			b, e := r.ReadBytes('\n')
			b = dropNewline(b)
			if len(b) > 0 {
				args = append(args, string(b))
			}
			if e != nil {
				if e != io.EOF {
					err = e
				}
				break
			}
		}
		if err != nil {
			Fatal(err)
		}
	}
	if len(args) == 0 {
		Fatal("error: no arguments")
	}

	var buf bytes.Buffer
	for i, arg := range args {
		buf.Reset()
		buf.Grow(len(arg) * 2)
		if i > 0 && !*noNewline {
			buf.WriteByte('\n')
		}
		a := strings.Split(arg, *splitSep)
		if *joinSep == "" {
			for _, s := range a {
				buf.WriteString(s)
				buf.WriteByte('\n')
			}
		} else {
			sep := *joinSep
			for j, s := range a {
				if j != 0 {
					buf.WriteString(sep)
				}
				buf.WriteString(s)
			}
			buf.WriteByte('\n')
		}
		if _, err := buf.WriteTo(os.Stdout); err != nil {
			Fatal(err)
		}
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
