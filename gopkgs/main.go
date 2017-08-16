package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type Ext struct {
	Ext   string
	Count int
}

type ByCount []Ext

func (b ByCount) Len() int           { return len(b) }
func (b ByCount) Less(i, j int) bool { return b[i].Count < b[j].Count }
func (b ByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type PkgFinder struct {
	ctxt   *build.Context
	pkgs   map[string]*build.Package
	seen   map[string]bool
	ignore map[string]bool
	paths  chan string
	pmu    sync.Mutex
	smu    sync.Mutex
	wg     sync.WaitGroup
}

func (w *PkgFinder) ValidName(s string) bool {
	return len(s) > 0 && s[0] != '_' && s[0] != '.' && !w.ignore[s]
}

func (*PkgFinder) HasGoFiles(path string) bool {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if DEBUG {
			fmt.Fprintf(os.Stderr, "opening file (%s): %s\n", path, err)
		}
		return false
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		if DEBUG {
			fmt.Fprintf(os.Stderr, "reading dirnames (%s): %s\n", path, err)
		}
		return false
	}
	for _, s := range names {
		if strings.HasSuffix(s, ".go") {
			return true
		}
	}
	return false
}

func (w *PkgFinder) Worker() {
	defer w.wg.Done()
	for path := range w.paths {
		pkg, err := w.ctxt.ImportDir(path, build.ImportComment)
		if err != nil {
			if DEBUG {
				fmt.Fprintf(os.Stderr, "error (%s): %s\n", path, err)
			}
			continue
		}
		w.pmu.Lock()
		w.pkgs[path] = pkg
		w.pmu.Unlock()
	}
}

func (w *PkgFinder) Walk(path string, fi os.FileInfo, err error) error {
	if !fi.IsDir() {
		return nil
	}
	if !w.ValidName(fi.Name()) {
		return filepath.SkipDir
	}
	w.smu.Lock()
	switch {
	case !w.seen[path]:
		w.seen[path] = true
		w.paths <- path
	case DEBUG:
		fmt.Fprintf(os.Stderr, "visited duplicate path: %s\n", path)
	}
	w.smu.Unlock()
	return nil
}

func IsDir(name string) bool {
	fi, err := os.Lstat(name)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
	return err == nil && fi.IsDir()
}

type PkgByImportPath []*build.Package

func (p PkgByImportPath) Len() int           { return len(p) }
func (p PkgByImportPath) Less(i, j int) bool { return p[i].ImportPath < p[j].ImportPath }
func (p PkgByImportPath) Swap(i, j int) {
	p[i].ImportPath, p[j].ImportPath = p[j].ImportPath, p[i].ImportPath
}

var (
	GOPATH  string
	DEBUG   bool
	IGNORE  string
	ALL     bool
	IMPORTS bool
)

func init() {
	flag.StringVar(&GOPATH, "GOPATH", build.Default.GOPATH, "Set GOPATH")

	flag.StringVar(&IGNORE, "ignore", "", "List of comma separated directories to "+
		"to ignore, by default 'vendor' and  'testdata' are ignored (unless -all is supplied)")

	flag.BoolVar(&DEBUG, "debug", false, "Print lots of debugging information")
	flag.BoolVar(&DEBUG, "d", false, "Print lots of debugging information (shorthand)")

	flag.BoolVar(&ALL, "all", false, "Include all directories (vendor and testdata)")
	flag.BoolVar(&ALL, "a", false, "Include all directories (vendor and testdata) (shorthand)")

	flag.BoolVar(&IMPORTS, "imports", false, "List all imports")
	flag.BoolVar(&IMPORTS, "i", false, "List all imports (shorthand)")
}

func Usage() {
	fmt.Println("List Go packages\nUsage: gopks [<flag> ...] <args> ...")
	os.Exit(1)
}

func buildIgnores(s string) map[string]bool {
	list := strings.Split(IGNORE, ",")
	m := make(map[string]bool, 2+len(list))
	for _, s := range list {
		m[strings.TrimSpace(s)] = true
	}
	if !ALL {
		m["vendor"] = true
		m["testdata"] = true
	}
	return m
}

func Run() error {
	build.Default.GOPATH = GOPATH
	w := &PkgFinder{
		ctxt:   &build.Default,
		pkgs:   make(map[string]*build.Package),
		seen:   make(map[string]bool),
		paths:  make(chan string, 100),
		ignore: buildIgnores(IGNORE),
	}

	// Start workers
	for i := 0; i < runtime.NumCPU(); i++ {
		w.wg.Add(1)
		go w.Worker()
	}

	for _, s := range flag.Args() {
		path, err := filepath.Abs(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping (%s): cannot find absolute path\n", s)
			continue
		}
		if !IsDir(path) {
			fmt.Fprintf(os.Stderr, "skipping (%s): not a directory\n", s)
			continue
		}
		if err := filepath.Walk(path, w.Walk); err != nil {
			fmt.Fprintf(os.Stderr, "walking (%s): %s\n", s, err)
		}
	}
	close(w.paths)
	w.wg.Wait()

	pkgs := make(PkgByImportPath, 0, len(w.pkgs))
	for _, p := range w.pkgs {
		pkgs = append(pkgs, p)
	}
	sort.Sort(pkgs)

	if IMPORTS {
		PrintImports(w)
		return nil
	}
	for _, p := range pkgs {
		fmt.Println(p.ImportPath)
	}
	return nil
}

func main() {
	flag.Parse()
	start := time.Now()

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Missing directory argument[s]")
		Usage()
	}
	if GOPATH != build.Default.GOPATH {
		if _, err := os.Lstat(GOPATH); err != nil {
			fmt.Fprintf(os.Stderr, "invalid GOPATH argument (%s): %s\n", GOPATH, err)
			os.Exit(1)
		}
	}

	if err := Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}

	if DEBUG {
		d := time.Since(start)
		fmt.Println("Runtime:", d)
	}
}

func PrintImports(w *PkgFinder) {
	start := time.Now()

	s := make([]string, 0, len(w.pkgs))
	for _, p := range w.pkgs {
		s = append(s, p.ImportPath)
		s = append(s, p.Imports...)
	}
	if len(s) == 0 {
		return
	}

	sort.Strings(s)
	fmt.Println(s[0])

	var i int
	for ; len(s[i]) == 0 || s[i][0] == '.'; i++ {
	}
	a := s[i]
	for ; i < len(s); i++ {
		if s[i] != a {
			a = s[i]
			fmt.Println(s[i])
		}
	}
	fmt.Println("Print Imports:", time.Since(start))

	if DEBUG {
		fmt.Println("Print Imports:", time.Since(start))
	}
}
