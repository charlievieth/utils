package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"text/tabwriter"

	"github.com/charlievieth/utils/godu/fastwalk"
)

type Size int64

const (
	kB = 1024
	mB = kB * kB
	gB = mB * mB
)

func (s Size) String() string {
	switch {
	case s < kB:
		return strconv.FormatInt(int64(s), 10)
	case s < mB:
		f := float64(s) / kB
		return strconv.FormatFloat(f, 'f', 1, 64) + "K"
	case s < gB:
		f := float64(s) / mB
		return strconv.FormatFloat(f, 'f', 1, 64) + "M"
	default:
		f := float64(s) / gB
		return strconv.FormatFloat(f, 'f', 1, 64) + "G"
	}
}

type Walker struct {
	Files int64
	Size  int64
}

func (w *Walker) Walk(path string, fi os.FileInfo) error {
	atomic.AddInt64(&w.Files, 1)
	if fi != nil {
		atomic.AddInt64(&w.Size, fi.Size())
	}
	return nil
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func errHandler(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
}

func realMain(dirs []string) error {
	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	for _, dir := range dirs {
		if !isDir(dir) {
			continue
		}
		var w Walker
		if err := fastwalk.Walk(dir, w.Walk, errHandler); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			continue
		}
		fmt.Fprintf(wr, "%s:\t%s\n", Size(w.Size), dir)
		// fmt.Printf("%s: %s\n", dir, Size(w.Size))
	}
	return wr.Flush()
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dirs...")
		os.Exit(1)
	}
	if err := realMain(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
