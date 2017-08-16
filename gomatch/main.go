package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	if len(os.Args) != 3 {
		Fatal(fmt.Sprintf("%s PATTERN NAME", os.Args[0]))
	}
	pattern := os.Args[1]
	name := os.Args[2]
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		Fatal(err)
	}
	fmt.Println(matched)
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
