package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func Readdirnames(name string) ([]string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: PATTERN [PATH...]\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	pattern := flag.Arg(0)
	if _, err := filepath.Match(pattern, ""); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	var paths []string
	if n := flag.NArg(); n > 1 {
		for i := 1; i < n; i++ {
			paths = append(paths, flag.Arg(i))
		}
	} else {
		paths = []string{"."}
	}

	var failed bool
	for _, path := range paths {
		names, err := Readdirnames(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			failed = true
			continue
		}
		for _, name := range names {
			if ok, _ := filepath.Match(pattern, name); ok {
				fmt.Println(filepath.Join(path, name))
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}
