package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"unicode/utf8"

	"golang.org/x/text/unicode/runenames"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage %s: RUNES...\n"+
			"Print the Unicode name of RUNES.\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		r := bufio.NewReader(os.Stdin)
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Fprintln(os.Stderr, "error:", err)
					os.Exit(1)
				}
				break
			}
			args = append(args, s)
		}
	}
	tw := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	for i, s := range args {
		if i > 0 && utf8.RuneCountInString(s) > 1 {
			fmt.Fprint(tw, "\n")
		}
		for _, r := range s {
			_, err := fmt.Fprintf(tw, "%q:\t%s (%U)\n", r, runenames.Name(r), r)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
