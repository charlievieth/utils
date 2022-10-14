package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/tabwriter"

	"github.com/charlievieth/fastwalk"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)
}

// // typedef enum { NONE, KILO, MEGA, GIGA, TERA, PETA, UNIT_MAX } unit_t;
// // int unitp [] = { NONE, KILO, MEGA, GIGA, TERA, PETA };

// const (
// 	KILO_2_SZ = 1024
// 	MEGA_2_SZ = 1024 * 1024
// 	GIGA_2_SZ = 1024 * 1024 * 1024
// 	TERA_2_SZ = 1024 * 1024 * 1024 * 1024
// 	PETA_2_SZ = 1024 * 1024 * 1024 * 1024 * 1024

// 	KILO_SI_SZ = 1000
// 	MEGA_SI_SZ = 1000 * 1000
// 	GIGA_SI_SZ = 1000 * 1000 * 1000
// 	TERA_SI_SZ = 1000 * 1000 * 1000 * 1000
// 	PETA_SI_SZ = 1000 * 1000 * 1000 * 1000 * 1000
// )

// var (
// 	vals_si    = [...]int64{1, KILO_SI_SZ, MEGA_SI_SZ, GIGA_SI_SZ, TERA_SI_SZ, PETA_SI_SZ}
// 	vals_base2 = [...]int64{1, KILO_2_SZ, MEGA_2_SZ, GIGA_2_SZ, TERA_2_SZ, PETA_2_SZ}
// )

// type Unit int

// const (
// 	UnitNone Unit = iota
// 	UnitKilo
// 	UnitMega
// 	UnitGiga
// 	UnitTera
// 	UnitPeta

// 	UnitExa
// 	UnitZetta
// 	UnitYotta
// 	UnitMax
// )

// // kB kilobyte
// // MB megabyte
// // GB gigabyte
// // TB terabyte
// // PB petabyte
// // EB exabyte
// // ZB zettabyte
// // YB yottabyte

// // RENAME
// func (u Unit) Base2() int64 {
// 	const N = 1024
// 	switch u {
// 	case UnitNone:
// 		return 1
// 	case UnitKilo:
// 		return N
// 	case UnitMega:
// 		return N * N
// 	case UnitGiga:
// 		return N * N * N
// 	case UnitTera:
// 		return N * N * N * N
// 	case UnitPeta:
// 		return N * N * N * N * N
// 	case UnitExa:
// 		return N * N * N * N * N * N
// 	// case UnitZetta:
// 	// 	return N * N * N * N * N * N * N
// 	// case UnitYotta:
// 	// 	return N * N * N * N * N * N * N * N
// 	default:
// 		panic(fmt.Sprintf("Unit(%d)", int64(u)))
// 	}
// }

// func (u Unit) SI() int64 {
// 	const N = 1000
// 	switch u {
// 	case UnitNone:
// 		return 1
// 	case UnitKilo:
// 		return N
// 	case UnitMega:
// 		return N * N
// 	case UnitGiga:
// 		return N * N * N
// 	case UnitTera:
// 		return N * N * N * N
// 	case UnitPeta:
// 		return N * N * N * N * N
// 	case UnitExa:
// 		return N * N * N * N * N * N
// 	// case UnitZetta:
// 	// 	return N * N * N * N * N * N * N
// 	// case UnitYotta:
// 	// 	return N * N * N * N * N * N * N * N
// 	default:
// 		panic(fmt.Sprintf("Unit(%d)", int64(u)))
// 	}
// }

// var unitp = [...]Unit{UnitNone, UnitKilo, UnitMega, UnitGiga, UnitTera, UnitPeta}

// func AdjustUnit(val float64) (float64, Unit) {
// 	valp := vals_base2 // WARN
// 	abval := math.Abs(val)
// 	var unit Unit
// 	var sz Unit
// 	if abval != 0 {
// 		sz = Unit(math.Ilogb(abval) / 10)
// 		fmt.Println("SZ:", sz, math.Ilogb(abval))
// 	}
// 	if sz < UnitMax {
// 		unit = unitp[sz]
// 		val /= float64(valp[sz])
// 	}
// 	return val, unit
// }

// func PrintHumanVal(n float64) {
// 	// n *= 512
// 	bytes, unit := AdjustUnit(n)

// 	// if (bytes == 0)
// 	// 	(void)printf("  0B");
// 	// else if (bytes > 10)
// 	// 	(void)printf("%3.0f%c", bytes, "BKMGTPE"[unit]);
// 	// else
// 	// 	(void)printf("%3.1f%c", bytes, "BKMGTPE"[unit]);

// 	if bytes == 0 {
// 		fmt.Println("  0B")
// 	} else if bytes > 10 {
// 		fmt.Printf("%3.0f%c\n", bytes, "BKMGTPE"[unit])
// 	} else {
// 		fmt.Printf("%3.1f%c\n", bytes, "BKMGTPE"[unit])
// 	}
// }

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
	Size    Size
	Match   GlobSet
	Exclude GlobSet
}

func dirEntrySize(path string, de fs.DirEntry) (int64, bool) {
	typ := de.Type()
	if typ == fs.ModeSymlink {
		fi, err := fastwalk.StatDirEntry(path, de)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
			}
			return 0, false
		}
		typ = fi.Mode().Type()
	}
	if typ.IsRegular() {
		info, err := de.Info()
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
			}
			return 0, false
		}
		return info.Size(), true
	}
	return 0, false
}

func (w *Walker) Walk(path string, de fs.DirEntry, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return nil
	}
	if size, ok := dirEntrySize(path, de); ok && size > 0 {
		atomic.AddInt64((*int64)(&w.Size), size)
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
	err := fastwalk.Walk(nil, path, w.Walk)
	return w.Size, err
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
			size, err := walkPath(path)
			if err != nil {
				log.Println("error:", err)
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

// TODO: add exclusion logic !!!
/*
type Exclude struct {
	names   []string
	namesRe []*regexp.Regexp
	paths   []string
	pathsRe []*regexp.Regexp
}

func (e *Exclude) Name(name string) bool {
	for _, pattern := range e.names {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	for _, re := range e.namesRe {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

func (e *Exclude) Path(path string) bool {
	for _, pattern := range e.paths {
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
	}
	for _, re := range e.pathsRe {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
*/

type dotWalker struct {
	// files sync.Map // WARN: top-level files
	m sync.Map

	// TODO: we could use a map since we pre-allocate it,
	// but theres a race if a file is added between reading
	// the directory and the call to fastwalk.Walk
	//
	// m map[string]*int64
}

// func pathRoot(path string) (string, bool) {
// 	for i := 0; i < len(path); i++ {
// 		if os.IsPathSeparator(path[i]) {
// 			return path[:i], true
// 		}
// 	}
// 	return path, false
// }

func pathRoot(path string) string {
	if len(path) > 2 && path[0] == '.' && os.IsPathSeparator(path[1]) {
		path = path[2:]
	}
	for i := 0; i < len(path); i++ {
		if os.IsPathSeparator(path[i]) {
			return path[:i]
		}
	}
	return path
}

func (w *dotWalker) Walk(path string, de fs.DirEntry, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return nil
	}
	if size, ok := dirEntrySize(path, de); ok && size > 0 {
		root := pathRoot(path)
		v, ok := w.m.Load(root)
		if !ok {
			v, _ = w.m.LoadOrStore(root, new(int64))
			log.Println("missing key:", root) // WARN
		}
		if atomic.AddInt64(v.(*int64), size) < 0 {
			log.Printf("overflow: %s", root)
		}
	}
	return nil
}

func (w *dotWalker) Run() error {
	names, err := readdirnames(".")
	if err != nil {
		return err
	}
	// TODO: use a regular map
	for _, name := range names {
		w.m.Store(name, new(int64))
	}

	// exitCode := 0
	if err := fastwalk.Walk(nil, ".", w.Walk); err != nil {
		log.Println("error: walk:", err)
		// exitCode = 1
	}

	var szs []FileSize
	w.m.Range(func(key, value interface{}) bool {
		szs = append(szs, FileSize{
			Name: key.(string),
			Size: Size(*value.(*int64)),
		})
		return true
	})

	sort.Sort(filesByName(szs))
	sort.Stable(filesBySize(szs))

	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.AlignRight)

	var total Size
	for _, sz := range szs {
		total += sz.Size
		if err := printSize(wr, sz.Name, sz.Size); err != nil {
			return err
		}
	}
	if err := printSize(wr, "total", total); err != nil {
		return err
	}
	if err := wr.Flush(); err != nil {
		return err
	}

	return nil
}

func walkDot() error {
	return new(dotWalker).Run()
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	flag.BoolVar(&fastwalk.DefaultConfig.Follow, "follow", false, "follow symbolic links")
	paths, flags := parseFlags()

	if *cpuprofile != "" {
		f, err := os.OpenFile(*cpuprofile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("could not create CPU profile:", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Println("could not start CPU profile:", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	const UseWalkDot = true

	// WARN: make sure that this is applicable for all cases
	// where filepath.Clean() == "."
	if len(paths) == 1 && (paths[0] == "." || filepath.Clean(paths[0]) == ".") {
		if UseWalkDot {
			if err := walkDot(); err != nil {
				log.Println("error:", err)
				os.Exit(1)
			}
			return
		} else {
			var err error
			paths, err = readdirnames(".")
			if err != nil {
				log.Println("error:", err)
				os.Exit(2)
			}
		}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	if err := walk(paths, flags); err != nil {
		log.Println("error:", err)
		os.Exit(2)
	}
}

/*
type Value interface {
    String() string
    Set(string) error
}
*/

type Glob struct {
	pattern string
	negate  bool
}

func (g Glob) String() string {
	return fmt.Sprintf("{Pattern: %q Negate: %t}", g.pattern, g.negate)
}

func NewGlob(s string) (*Glob, error) {
	negate := strings.HasPrefix(s, "!")
	s = strings.TrimPrefix(s, "!")
	if s == "" {
		return nil, errors.New("empty pattern")
	}
	if _, err := filepath.Match(s, ""); err != nil {
		return nil, err
	}
	return &Glob{pattern: s, negate: negate}, nil
}

func (g *Glob) Match(name string) bool {
	ok, err := filepath.Match(g.pattern, name)
	return err == nil && ok == !g.negate
}

type GlobSet struct {
	globs []*Glob

	// match   []string
	// exclude []string
}

// func (g *GlobSet) String() string {
// 	return fmt.Sprintf("%q", g.patterns)
// }

func (gs *GlobSet) Set(s string) error {
	g, err := NewGlob(s)
	if err != nil {
		return err
	}
	gs.globs = append(gs.globs, g)
	return nil

	// if s == "" {
	// 	return errors.New("empty pattern")
	// }
	// if _, err := filepath.Match(s, ""); err != nil {
	// 	return err
	// }
	// g.patterns = append(g.patterns, s)
	// return nil
}

func (g *GlobSet) Match(name string) bool {
	if g == nil || len(g.globs) == 0 {
		return true
	}
	for _, g := range g.globs {
		if !g.Match(name) {
			return false
		}
	}
	return false

	// if g == nil || len(g.patterns) == 0 {
	// 	return true
	// }
	// for _, p := range g.patterns {
	// 	if ok, _ := filepath.Match(p, name); ok {
	// 		return true
	// 	}
	// }
	// return false
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
