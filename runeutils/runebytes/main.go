package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Usage = func() {
		const usage = "%s: [string...]\n" +
			"Print the bytes of the UTF-8 encoded form of each run in the input string.\n"
		fmt.Fprintf(flag.CommandLine.Output(), usage, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	fileName := flag.String("f", "", "read data from file")
	flag.Parse()

	var args []string
	if *fileName != "" || flag.NArg() == 0 || flag.Arg(0) == "-" {
		var data []byte
		if *fileName != "" {
			b, err := os.ReadFile(*fileName)
			if err != nil {
				log.Fatal(err)
			}
			data = b
		} else {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			data = b
		}
		args = strings.Fields(string(data))
	} else {
		args = flag.Args()
	}
	tw := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	for _, arg := range args {
		for _, r := range arg {
			s := string(r)
			for i := 0; i < len(s); i++ {
				fmt.Fprintf(tw, "%d\t", s[i])
			}
			for i := len(s); i < 4; i++ {
				fmt.Fprint(tw, "0\t")
			}
			if _, err := fmt.Fprintf(tw, "|\t%q\n", s); err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := tw.Flush(); err != nil {
		log.Fatal(err)
	}
}
