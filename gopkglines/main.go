package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Types from: "go help list"

type ModuleError struct {
	Err string // the error itself
}

type Module struct {
	Path       string       // module path
	Query      string       // version query corresponding to this version
	Version    string       // module version
	Versions   []string     // available module versions
	Replace    *Module      // replaced by this module
	Time       *time.Time   // time version was created
	Update     *Module      // available update (with -u)
	Main       bool         // is this the main module?
	Indirect   bool         // module is only indirectly needed by main module
	Dir        string       // directory holding local copy of files, if any
	GoMod      string       // path to go.mod file describing module, if any
	GoVersion  string       // go version used in module
	Retracted  []string     // retraction information, if any (with -retracted or -u)
	Deprecated string       // deprecation message, if any (with -u)
	Error      *ModuleError // error loading module
	Origin     any          // provenance of module
	Reuse      bool         // reuse of old module info is safe
}

type PackageError struct {
	ImportStack []string // shortest path from package named on command line to this one
	Pos         string   // position of error (if present, file:line:col)
	Err         string   // the error itself
}

type Package struct {
	Dir            string   // directory containing package sources
	ImportPath     string   // import path of package in dir
	ImportComment  string   // path in import comment on package statement
	Name           string   // package name
	Doc            string   // package documentation string
	Target         string   // install path
	Shlib          string   // the shared library that contains this package (only set when -linkshared)
	Goroot         bool     // is this package in the Go root?
	Standard       bool     // is this package part of the standard Go library?
	Stale          bool     // would 'go install' do anything for this package?
	StaleReason    string   // explanation for Stale==true
	Root           string   // Go root or Go path dir containing this package
	ConflictDir    string   // this directory shadows Dir in $GOPATH
	BinaryOnly     bool     // binary-only package (no longer supported)
	ForTest        string   // package is only for use in named test
	Export         string   // file containing export data (when using -export)
	BuildID        string   // build ID of the compiled package (when using -export)
	Module         *Module  // info about package's containing module, if any (can be nil)
	Match          []string // command-line patterns matching this package
	DepOnly        bool     // package is only a dependency, not explicitly listed
	DefaultGODEBUG string   // default GODEBUG setting, for main packages

	// Source files
	GoFiles           []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles          []string // .go source files that import "C"
	CompiledGoFiles   []string // .go files presented to compiler (when using -compiled)
	IgnoredGoFiles    []string // .go source files ignored due to build constraints
	IgnoredOtherFiles []string // non-.go source files ignored due to build constraints
	CFiles            []string // .c source files
	CXXFiles          []string // .cc, .cxx and .cpp source files
	MFiles            []string // .m source files
	HFiles            []string // .h, .hh, .hpp and .hxx source files
	FFiles            []string // .f, .F, .for and .f90 Fortran source files
	SFiles            []string // .s source files
	SwigFiles         []string // .swig files
	SwigCXXFiles      []string // .swigcxx files
	SysoFiles         []string // .syso object files to add to archive
	TestGoFiles       []string // _test.go files in package
	XTestGoFiles      []string // _test.go files outside package

	// Embedded files
	EmbedPatterns      []string // //go:embed patterns
	EmbedFiles         []string // files matched by EmbedPatterns
	TestEmbedPatterns  []string // //go:embed patterns in TestGoFiles
	TestEmbedFiles     []string // files matched by TestEmbedPatterns
	XTestEmbedPatterns []string // //go:embed patterns in XTestGoFiles
	XTestEmbedFiles    []string // files matched by XTestEmbedPatterns

	// Cgo directives
	CgoCFLAGS    []string // cgo: flags for C compiler
	CgoCPPFLAGS  []string // cgo: flags for C preprocessor
	CgoCXXFLAGS  []string // cgo: flags for C++ compiler
	CgoFFLAGS    []string // cgo: flags for Fortran compiler
	CgoLDFLAGS   []string // cgo: flags for linker
	CgoPkgConfig []string // cgo: pkg-config names

	// Dependency information
	Imports      []string          // import paths used by this package
	ImportMap    map[string]string // map from source import to ImportPath (identity entries omitted)
	Deps         []string          // all (recursively) imported dependencies
	TestImports  []string          // imports from TestGoFiles
	XTestImports []string          // imports from XTestGoFiles

	// Error information
	Incomplete bool            // this package or a dependency has an error
	Error      *PackageError   // error loading package
	DepsErrors []*PackageError // errors loading dependencies
}

func LineCount(name string) (int, error) {
	src, err := os.ReadFile(name)
	if err != nil {
		return 0, err
	}
	return bytes.Count(src, []byte{'\n'}), nil
}

func Dependencies(pkgName string) []string {
	out, err := exec.Command("go", "list", "-json", pkgName).CombinedOutput()
	if err != nil {
		log.Fatalf("%s: %s", err, out)
	}
	var pkg Package
	if err := json.Unmarshal(out, &pkg); err != nil {
		log.Fatal(err)
	}
	a := pkg.Imports[:0]
	for _, s := range pkg.Imports {
		if strings.HasPrefix(s, "github.com/prometheus/") {
			a = append(a, s)
		}
	}
	a = append(a, pkgName)
	sort.Strings(a) // make this consistent
	return a
}

func RequirePackage(name string) {
	path := filepath.Join(build.Default.GOPATH, "src", name)
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("missing required package %q - please Git clone it to: %q", name, path)
	}
	vendor := filepath.Join(build.Default.GOPATH, "src", name, "vendor")
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("missing vendor directory %q - please run 'go mod vendor' in package: %q", vendor, name)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-testfiles] GO_PKG_NAME\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	testfiles := flag.Bool("testfiles", false, "include Go test files")
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		log.Fatal("missing required argument: GO_PKG_NAME")
	}
	pkgName := flag.Arg(0)

	RequirePackage(pkgName)

	out, err := exec.Command("go", append([]string{"list", "-json"}, Dependencies()...)...).CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	var pkgs []Package
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var pkg Package
		if err := dec.Decode(&pkg); err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}
		pkgs = append(pkgs, pkg)
	}

	var paths []string
	for _, p := range pkgs {
		for _, list := range [][]string{p.GoFiles, p.CgoFiles, p.CFiles, p.CXXFiles} {
			for _, name := range list {
				paths = append(paths, filepath.Join(p.Dir, name))
			}
		}
		if *testfiles {
			for _, name := range p.TestGoFiles {
				paths = append(paths, filepath.Join(p.Dir, name))
			}
		}
	}

	sort.Strings(paths) // make output consistent

	var root string
	for _, p := range pkgs {
		if p.Root != "" {
			root = p.Root
			break
		}
	}

	total := 0
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 2, ' ', 0)
	for _, path := range paths {
		lines, err := LineCount(path)
		if err != nil {
			log.Fatal(err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "%d:\t%s\n", lines, rel)
		total += lines
	}
	fmt.Fprintf(w, "%d:\ttotal\n", total)
	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}
}

// func PrintJSON(v interface{}) {
// 	enc := json.NewEncoder(os.Stdout)
// 	enc.SetIndent("", "    ")
// 	if err := enc.Encode(v); err != nil {
// 		log.Fatal(err)
// 	}
// }
