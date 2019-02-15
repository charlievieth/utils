package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const DefaultTimestamp = "15:04:05.000000"

func main() {
	var parseDur bool
	flag.BoolVar(&parseDur, "dur", false, "Calculate time since [TIMESTAMP]")
	flag.BoolVar(&parseDur, "d", false, "Calculate time since [TIMESTAMP] (shorthand)")
	flag.Parse()

	if parseDur {
		if flag.NArg() != 1 {
			Fatal(fmt.Sprintf("USAGE: %s -dur TIMESTAMP", filepath.Base(os.Args[0])))
		}
		// this is lazy, but whatever
		now, err := time.Parse(DefaultTimestamp, time.Now().Format(DefaultTimestamp))
		if err != nil {
			Fatal(err) // this should never happen
		}
		t, err := time.Parse(DefaultTimestamp, flag.Arg(0))
		if err != nil {
			Fatal(err)
		}
		fmt.Println(now.Sub(t).String())
		return
	}

	fmt.Println(time.Now().Format(DefaultTimestamp))
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var s string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		s = "Error"
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
	}
	os.Exit(1)
}
