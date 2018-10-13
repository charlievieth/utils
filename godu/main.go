package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"text/tabwriter"

	"github.com/charlievieth/utils/godu/fastwalk"
)

const SectorSize = 4096

func RoundUp(x int64) int64 {
	return ((x + SectorSize - 1) / SectorSize) * SectorSize
}

type Size int64

const (
	kB = 1024
	mB = kB * 1024
	gB = mB * 1024
	tB = gB * 1024
	pB = tB * 1024
)

func (s Size) String() string {
	// TODO: test s%SectorSize == 0 HERE and optimize
	switch {
	case s < kB:
		return strconv.FormatInt(int64(s), 10) + "B"
	case s < mB:
		if s < 10*kB {
			f := float64(s) / kB
			return strconv.FormatFloat(f, 'f', 1, 64) + "K"
		}
		return strconv.FormatInt(int64(s)/kB, 10) + "k"
	case s < gB:
		if s < 10*mB {
			f := float64(s) / mB
			return strconv.FormatFloat(f, 'f', 1, 64) + "M"
		}
		return strconv.FormatInt(int64(s)/mB, 10) + "M"
	case s < tB:
		if s < 10*gB {
			f := float64(s) / gB
			return strconv.FormatFloat(f, 'f', 1, 64) + "G"
		}
		return strconv.FormatInt(int64(s)/gB, 10) + "G"
	case s < pB:
		if s < 10*tB {
			f := float64(s) / tB
			return strconv.FormatFloat(f, 'f', 1, 64) + "T"
		}
		return strconv.FormatInt(int64(s)/tB, 10) + "T"
	default:
		if s < 10*pB {
			f := float64(s) / pB
			return strconv.FormatFloat(f, 'f', 1, 64) + "P"
		}
		return strconv.FormatInt(int64(s)/pB, 10) + "P"
	}
}

type FileSize struct {
	Size Size
	Name string
}

type filesBySize []FileSize

func (f filesBySize) Len() int           { return len(f) }
func (f filesBySize) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f filesBySize) Less(i, j int) bool { return f[i].Size < f[j].Size }

type filesByName []FileSize

func (f filesByName) Len() int           { return len(f) }
func (f filesByName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f filesByName) Less(i, j int) bool { return f[i].Name < f[j].Name }

type filesByNameCase []FileSize

func (f filesByNameCase) Len() int      { return len(f) }
func (f filesByNameCase) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f filesByNameCase) Less(i, j int) bool {
	return strings.ToLower(f[i].Name) < strings.ToLower(f[j].Name)
}

type Walker struct {
	Size Size
}

func (w *Walker) Walk(path string, fi os.FileInfo) error {
	if fi != nil && fi.Mode().IsRegular() {
		atomic.AddInt64((*int64)(&w.Size), fi.Size())
	}
	return nil
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func printSize(wr *tabwriter.Writer, name string, size Size) error {
	_, err := fmt.Fprintf(wr, "%s\t    %s\n", size, name)
	return err
}

func walkPath(path string) (Size, error) {
	if !isDir(path) {
		fi, err := os.Lstat(path)
		if err != nil {
			return 0, err
		}
		return Size(fi.Size()), nil
	}
	var w Walker
	return w.Size, fastwalk.Walk(path, w.Walk, func(err error) {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	})
}

func walk(paths []string, flags uint) error {
	var sizes []FileSize
	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight)
	for _, path := range paths {
		size, err := walkPath(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			continue
		}
		if flags&PrintBytes == 0 {
			size = Size(RoundUp(int64(size)))
		}
		if flags&SortSize != 0 {
			sizes = append(sizes, FileSize{size, path})
			continue
		}
		if err := printSize(wr, path, size); err != nil {
			return err // can't print pipe may be broken
		}
	}
	if flags&SortSize == 0 {
		return wr.Flush()
	}
	if flags&SortNameCase != 0 {
		sort.Sort(filesByNameCase(sizes))
	} else {
		sort.Sort(filesByName(sizes))
	}
	sort.Stable(filesBySize(sizes))
	for _, sz := range sizes {
		if err := printSize(wr, sz.Name, sz.Size); err != nil {
			return err
		}
	}
	return wr.Flush()
}

const (
	SortSize uint = 1 << iota
	SortNameCase
	PrintBytes
)

func parseFlags() uint {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE %s: [OPTION]... [FILE]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	bySize := flag.Bool("s", false, "Sort files by size")
	bySizeCase := flag.Bool("sc", false, "Sort files by size and name case-insensitivity")
	printBytes := flag.Bool("b", false, "Print apparent size in bytes rather than disk usage")

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var flags uint
	if bySize != nil && *bySize {
		flags |= SortSize
	}
	if bySizeCase != nil && *bySizeCase {
		flags |= SortSize
		flags |= SortNameCase
	}
	if printBytes != nil && *printBytes {
		flags |= PrintBytes
	}

	return flags
}

func main() {
	flags := parseFlags()
	if err := walk(flag.Args(), flags); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(2)
	}
}
