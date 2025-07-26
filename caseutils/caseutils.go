package caseutils

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ProgMain(upper bool) {
	flag.Usage = func() {
		const usage = "Usage: %[1]s [TEXT]...\n" +
			"Convert provided text to %[2]s-case. If no text is provided STDIN\n" +
			"is read and converted to %[2]s-case."
		caseStr := "upper"
		if !upper {
			caseStr = "lower"
		}
		fmt.Fprintf(flag.CommandLine.Output(), usage, filepath.Base(os.Args[0]), caseStr)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 0 {
		for _, s := range flag.Args() {
			if upper {
				s = strings.ToUpper(s)
			} else {
				s = strings.ToLower(s)
			}
			fmt.Println(s)
		}
		return
	}

	// STDIN
	r := bufio.NewReader(os.Stdin)
	for {
		b, err := r.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "ERROR:", err)
				os.Exit(1)
			}
			break
		}
		if upper {
			b = bytes.ToUpper(b)
		} else {
			b = bytes.ToLower(b)
		}
		if _, err := os.Stdout.Write(b); err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
	}
}
