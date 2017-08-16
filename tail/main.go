package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

var LineCount int

func init() {
	flag.IntVar(&LineCount, "lines", -1, "output the last NUM lines")
	flag.IntVar(&LineCount, "n", -1, "output the last NUM lines")
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		Fatal("missing filename arg")
	}
}

func Fatal(err interface{}) {
	var s string
	if _, file, line, ok := runtime.Caller(1); ok {
		s = fmt.Sprintf("%s:%d", file, line)
	}
	if err != nil {
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
