package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func streamStdin() (err error) {
	rd := bufio.NewReader(os.Stdin)
	for {
		r, _, e := rd.ReadRune()
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
		if r != '\x00' && r != '\n' {
			fmt.Printf("%d: %q\n", utf8.RuneLen(r), r)
		}
	}
	return err
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: [RUNES]...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 || flag.Arg(0) == "-" {
		if err := streamStdin(); err != nil {
			fmt.Fprintf(os.Stdin, "error: %s\n", err)
			os.Exit(1)
		}
	}

	for _, arg := range flag.Args() {
		for _, a := range strings.Split(arg, "") {
			for _, r := range a {
				fmt.Printf("%d: %q\n", utf8.RuneLen(r), r)
			}
		}
	}
}
