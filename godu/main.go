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
	mB = kB * 1024
	gB = mB * 1024
)

func (s Size) String() string {
	// TODO: test s%SectorSize == 0 HERE and optimize
	switch {
	case s < kB:
		return strconv.FormatInt(int64(s), 10) + "B"
	case s < mB:
		if s%SectorSize == 0 {
			return strconv.FormatInt(int64(s)/kB, 10) + "k"
		}
		f := float64(s) / kB
		return strconv.FormatFloat(f, 'f', 1, 64) + "K"
	case s < gB:
		if s%SectorSize == 0 {
			return strconv.FormatInt(int64(s)/mB, 10) + "M"
		}
		f := float64(s) / mB
		return strconv.FormatFloat(f, 'f', 1, 64) + "M"
	default:
		if s%SectorSize == 0 {
			return strconv.FormatInt(int64(s)/gB, 10) + "G"
		}
		f := float64(s) / gB
		return strconv.FormatFloat(f, 'f', 1, 64) + "G"
	}
}

func (s *Size) Add(n int64) { atomic.AddInt64((*int64)(s), n) }

type Walker struct {
	Files int64
	Size  Size
}

const SectorSize = 4096

func RoundUp(x int64) int64 {
	return ((x + SectorSize - 1) / SectorSize) * SectorSize
}

func (w *Walker) Walk(path string, fi os.FileInfo) error {
	atomic.AddInt64(&w.Files, 1)
	if fi != nil && !fi.IsDir() {
		w.Size.Add(RoundUp(fi.Size()))
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
	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight)
	for _, dir := range dirs {
		if !isDir(dir) {
			continue
		}
		var w Walker
		if err := fastwalk.Walk(dir, w.Walk, errHandler); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			continue
		}
		fmt.Fprintf(wr, "%s\t  %s\n", Size(w.Size), dir)
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
