package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
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

func (w *Walker) Walk(path string, typ os.FileMode) error {
	if typ.IsRegular() {
		if size, err := GetFileSize(path); err == nil {
			atomic.AddInt64((*int64)(&w.Size), size)
		}
	}
	return nil
}

func (w *Walker) WalkSize(fd int, _, basename string) {
	if n, _ := GetFileSizeAt(fd, basename, false); n != 0 {
		atomic.AddInt64((*int64)(&w.Size), n)
	}
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

func ErrFunc(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
}

func walkPathSize(path string) (Size, error) {
	if !isDir(path) {
		fi, err := os.Lstat(path)
		if err != nil {
			return 0, err
		}
		return Size(fi.Size()), nil
	}
	var w Walker
	return w.Size, fastwalk.WalkSize(path, w.WalkSize, ErrFunc)
}

func gateSize() int {
	// TODO: consider using "/ 3" on darwin
	n := runtime.NumCPU() / 2
	// handle the fact that darwin has a garbage FS
	if runtime.GOOS == "darwin" {
		if n >= 12 {
			n = 12
		}
	}
	if n == 0 {
		n = 2
	}
	return n
}

func walk(paths []string, flags uint) error {
	var (
		walkErr error
		total   Size
		sizes   []FileSize
		mu      sync.Mutex
		wg      sync.WaitGroup
	)
	if flags&SortSize != 0 {
		sizes = make([]FileSize, 0, len(paths))
	}
	stop := make(chan struct{})
	gate := make(chan struct{}, gateSize()) // limit the number of parallel walks

	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight)

Loop:
	for _, path := range paths {
		select {
		case gate <- struct{}{}:
			// Ok
		case <-stop:
			break Loop
		}
		wg.Add(1)
		go func(path string) {
			defer func() {
				<-gate
				wg.Done()
			}()
			// WARN WARN WARN
			// WARN WARN WARN
			// size, err := walkPath(path)
			size, err := walkPathSize(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				return
			}
			if flags&PrintBytes == 0 {
				size = Size(RoundUp(int64(size)))
			}
			mu.Lock()
			defer mu.Unlock()
			total += size
			if flags&SortSize != 0 {
				sizes = append(sizes, FileSize{size, path})
				return
			}
			if err := printSize(wr, path, size); err != nil {
				if walkErr == nil {
					walkErr = err
					close(stop)
				}
			}
		}(path)
	}
	wg.Wait()

	if walkErr != nil {
		return walkErr
	}

	if flags&SortSize != 0 {
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
	}
	if flags&PrintTotal != 0 {
		if err := printSize(wr, "total", total); err != nil {
			return err
		}
	}
	return wr.Flush()
}

const (
	SortSize uint = 1 << iota
	SortNameCase
	PrintBytes
	PrintTotal
)

func parseFlags() ([]string, uint) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE %s: [OPTION]... [FILE]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	bySize := flag.Bool("s", false, "Sort files by size")
	bySizeCase := flag.Bool("sc", false, "Sort files by size and name case-insensitivity")
	printBytes := flag.Bool("b", false, "Print apparent size in bytes rather than disk usage")
	printTotal := flag.Bool("c", false, "Display a grand total")

	flag.Parse()

	var flags uint
	if *bySize {
		flags |= SortSize
	}
	if *bySizeCase {
		flags |= SortSize
		flags |= SortNameCase
	}
	if *printBytes {
		flags |= PrintBytes
	}
	if *printTotal {
		flags |= PrintTotal
	}

	return flag.Args(), flags
}

func readdirnames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	return names, err
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	paths, flags := parseFlags()

	if *cpuprofile != "" {
		f, err := os.OpenFile(*cpuprofile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not create CPU profile: ", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintln(os.Stderr, "could not start CPU profile: ", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	if len(paths) == 1 && paths[0] == "." {
		var err error
		paths, err = readdirnames(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(2)
		}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	if err := walk(paths, flags); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(2)
	}
}

/*
	errCh := make(chan error, 1)
	stop := make(chan struct{})
	gate := make(chan struct{}, gateSize()) // limit the number of parallel walks

	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, path := range paths {
			select {
			case gate <- struct{}{}:
				wg.Add(1)
				go func(path string) {
					defer func() {
						<-gate
						wg.Done()
					}()
					size, err := walkPath(path)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: %s\n", err)
						return
					}
					if flags&PrintBytes == 0 {
						size = Size(RoundUp(int64(size)))
					}
					mu.Lock()
					defer mu.Unlock()
					total += size
					if flags&SortSize != 0 {
						sizes = append(sizes, FileSize{size, path})
						return
					}
					if err := printSize(wr, path, size); err != nil {
						// can't print pipe may be broken
						select {
						case errCh <- err:
						default:
						}
					}
				}(path)
			case <-stop:
				return
			}
		}
	}()
	wgCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgCh)
	}()
	select {
	case <-wgCh:
		// Ok
	case err := <-errCh:
		close(stop)

		return err
	}
*/
