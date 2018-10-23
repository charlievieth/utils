package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"text/tabwriter"
)

var LineCount int
var MaxLength int

func init() {
	flag.IntVar(&LineCount, "n", 2, "Number of lines to transpose")
	flag.IntVar(&MaxLength, "l", -1, "Max line length, -1 means no max length")
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		Fatal("USAGE: [OPTIONS] FILENAME")
	}
	if LineCount <= 0 {
		Fatal("lines argument '-n' must be greater than 0")
	}
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	var lines [][]string
	for i := 0; i < LineCount; i++ {
		a, err := r.Read()
		if err != nil {
			Fatal(err)
		}
		lines = append(lines, a)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	for j := range lines[0] {
		for i := range lines {
			// TODO: don't trim the first column
			line := lines[i][j]
			if n := MaxLength; n > 0 && len(line) > n {
				if n > 6 {
					line = line[:n-len("...")] + "..."
				} else {
					line = line[:n]
				}
			}
			if i == 0 {
				fmt.Fprintf(w, "%s:", line)
			} else {
				fmt.Fprintf(w, "\t%s", line)
			}
		}
		fmt.Fprint(w, "\n")
	}
	if err := w.Flush(); err != nil {
		Fatal(err)
	}
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
	}
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
