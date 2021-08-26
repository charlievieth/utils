package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	// TODO: add support for capping line length
	flag.Usage = func() {
		const msg = "Usage: %[1]s [OPTION]... [ARGUMENTS]...\n" +
			"Hex encode STDIN or ARGUMENTS\n" +
			"\n" +
			"With no ARGUMENTS, or when the first ARGUMENTS is -, read standard input.\n"
		fmt.Fprintf(os.Stdout, msg, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if n := flag.NArg(); n > 0 && flag.Arg(0) != "-" {
		var b []byte
		for _, s := range flag.Args() {
			i := hex.EncodedLen(len(s))
			if len(b) < i {
				b = make([]byte, i+1)
			}
			hex.Encode(b, []byte(s))
			b[i] = '\n'
			if _, err := os.Stdout.Write(b); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
		return
	}

	w := hex.NewEncoder(os.Stdout)
	if _, err := io.Copy(w, os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
