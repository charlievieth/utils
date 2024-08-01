package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func main() {
	flag.Usage = func() {
		const usage = "%s: [string...]\n" +
			"Print the bytes of the UTF-8 encoded form of each run in the input string.\n"
		fmt.Fprintf(flag.CommandLine.Output(), usage, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	tw := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	for _, arg := range flag.Args() {
		for _, r := range arg {
			s := string(r)
			for i := 0; i < len(s); i++ {
				fmt.Fprintf(tw, "%d\t", s[i])
			}
			for i := len(s); i < 4; i++ {
				fmt.Fprint(tw, "0\t")
			}
			fmt.Fprintf(tw, "|\t%q\n", s)
		}
	}
	if err := tw.Flush(); err != nil {
		panic(err)
	}
}
