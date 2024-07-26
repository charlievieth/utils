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

// NB: completely forgot I can use 'wc -c' for this
func main() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stderr)
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "%s: print the byte size of UTF-8 strings\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 || flag.Arg(0) == "-" {
		// Just read the number of bytes
		buf := make([]byte, 32*1024)
		sz := 0
		for {
			n, err := os.Stdin.Read(buf)
			sz += n
			if err != nil {
				if err != io.EOF {
					log.Fatal(err)
				}
				break
			}
		}
		fmt.Println(sz)
		return
	}

	const esc = string(tabwriter.Escape)
	r := strings.NewReplacer("\t", esc+"\t"+esc, "\n", esc+"\n"+esc)

	w := tabwriter.NewWriter(os.Stdout, 4, 8, 2, ' ', 0)
	for _, s := range flag.Args() {
		fmt.Fprintf(w, "%d:\t%s\n", len(s), r.Replace(s))
	}
	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}
}
