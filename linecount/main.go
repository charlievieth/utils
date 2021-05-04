package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/charlievieth/num"
	"github.com/charlievieth/pkgs/fastwalk"
)

// var xKnownFileNames = map[string]string{
// 	"AUTHORS":   "AUTHORS",
// 	"BACKUP":    "BACKUP",
// 	"BASEIMAGE": "BASEIMAGE",
// 	"BUILD":     "BUILD",
// 	// "BUILD.bazel":     "BUILD.bazel",
// 	"ChangeLog":       "CHANGELOG",
// 	"CHANGELOG":       "CHANGELOG",
// 	"CHANGELOG.md":    "CHANGELOG",
// 	"CMakeLists.txt":  "CMakeLists.txt",
// 	"CODEOWNERS":      "CODEOWNERS",
// 	"CONTRIBUTING":    "CONTRIBUTING",
// 	"CONTRIBUTING.md": "CONTRIBUTING",
// 	"CONTRIBUTORS":    "CONTRIBUTORS",
// 	"COPYING":         "COPYING",
// 	"Depend":          "Depend",
// 	"Dockerfile":      "Dockerfile",
// 	"Doxyfile":        "Doxyfile",
// 	"Gemfile":         "Gemfile",
// 	"GNUmakefile":     "GNUmakefile",
// 	"GOVERNANCE.md":   "GOVERNANCE",
// 	"Implies":         "Implies",
// 	"INSTALL":         "INSTALL",
// 	"INSTALLER":       "INSTALLER",
// 	"LICENSE":         "LICENSE",
// 	"LICENSE.md":      "LICENSE",
// 	"LICENSE.txt":     "LICENSE",
// 	"LINGUAS":         "LINGUAS",
// 	"MAINTAINERS":     "MAINTAINERS",
// 	"MAINTAINERS.md":  "MAINTAINERS",
// 	"makefile":        "Makefile",
// 	"Makefile":        "Makefile",
// 	"MANIFEST":        "MANIFEST",
// 	"METADATA":        "METADATA",
// 	"mkinstalldirs":   "mkinstalldirs",
// 	"NEWS":            "NEWS",
// 	"NOTICE":          "NOTICE",
// 	"OWNERS":          "OWNERS",
// 	"PATENTS":         "PATENTS",
// 	"Podfile":         "Podfile",
// 	"Rakefile":        "Rakefile",
// 	"README":          "README",
// 	"README.md":       "README",
// 	"README.txt":      "README",
// 	"RECORD":          "RECORD",
// 	"TODO":            "TODO",
// 	"TODO.md":         "TODO",
// 	"TODO.txt":        "TODO",
// 	"VERSION":         "VERSION",
// 	"Versions":        "Versions",
// 	"WHEEL":           "WHEEL",
// 	"WORKSPACE":       "WORKSPACE",
// }

func WellKnownFilename(s string) bool {
	switch s {
	case "Dockerfile", "Gemfile", "Makefile", "Podfile", "Rakefile",
		"CMakeLists.txt", "LICENSE", "MANIFEST", "METADATA", "NOTICE",
		"AUTHORS", "CODEOWNERS", "CONTRIBUTORS", "README", "PATENTS",
		"OWNERS", "BUILD", "WORKSPACE", "tags":
		return true
	}
	return false
}

func IgnoredExtension(ext string) bool {
	switch ext {
	case ".bz", ".bzip", ".exe", ".gz", ".gzip", ".tar", ".tbz", ".tgz",
		".vdi", ".xz", ".zip", ".zst":
		return true
	}
	return false
}

func Ext(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case "":
		base := filepath.Base(path)
		if WellKnownFilename(base) {
			ext = base
		}
	case ".txt":
		if strings.HasSuffix(path, "CMakeLists.txt") {
			ext = "CMakeLists.txt"
		}
	}
	return ext
}

func ExecutableMode(m os.FileMode) bool {
	const mask = 1 | 8 | 64
	return m&mask != 0
}

// Tested with 16 and 32k and 8k seems best
const bufSize = 8 * 1024

var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, bufSize)
		return &b
	},
}

func isBinary(b []byte) bool {
	if len(b) > 512 {
		b = b[:512]
	}
	return bytes.IndexByte(b, 0) != -1
}

var ErrBinary = errors.New("binary file")

var newLine = []byte{'\n'}

func LineCount(filename string, needExt bool) (int64, string, error) {

	f, err := os.Open(filename)
	if err != nil {
		return 0, "", err
	}
	p := bufPool.Get().(*[]byte)
	defer func() {
		f.Close()
		bufPool.Put(p)
	}()
	buf := *p

	var ext string // TODO: "exe" or "ext" ???
	var lines int64

	first := true
	for {
		nr, er := f.Read(buf)
		if first {
			if isBinary(buf[0:nr]) {
				return 0, "", ErrBinary
			}
			first = false
		}
		lines += int64(bytes.Count(buf[0:nr], newLine))
		if needExt && ext == "" {
			ext = ExtractShebang(buf[0:nr])
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return lines, ext, err
}

type Walker struct {
	mu       sync.Mutex
	exts     map[string]int64
	ignore   map[string]bool
	seen     SeenFiles
	symlinks bool
}

func (w *Walker) Walk(path string, typ os.FileMode) error {
	if typ.IsRegular() {
		if ExecutableMode(typ) {
			return nil
		}
		ext := Ext(path)
		if IgnoredExtension(ext) {
			return nil
		}
		lines, scriptExt, err := LineCount(path, ext == "")
		if err != nil {
			if err != ErrBinary {
				return err
			}
			return nil
		}
		if ext == "" && scriptExt != "" {
			// WARN: debug only
			// fmt.Fprintf(os.Stderr, "%s => %s\n", scriptExt, path)
			ext = scriptExt + "-script"
		}
		w.mu.Lock()
		w.exts[ext] += lines
		w.mu.Unlock()
		return nil
	}
	if typ == os.ModeDir {
		base := filepath.Base(path)
		if base == "" || base[0] == '.' || base[0] == '_' ||
			base == "testdata" || base == "node_modules" || base == "venv" {
			return filepath.SkipDir
		}
		if w.ignore[base] {
			return filepath.SkipDir
		}
		return nil
	}
	return nil
}

func (w *Walker) WalkLinks(path string, typ os.FileMode) error {
	if typ&os.ModeSymlink != 0 {
		fi, err := os.Stat(path)
		if err != nil {
			// handle
			return nil
		}
		typ = fi.Mode()
	}
	seen := w.seen.Path(path)
	if typ.IsRegular() {
		if seen {
			return nil
		}
		ext := Ext(path)
		if IgnoredExtension(ext) {
			return nil
		}
		lines, scriptExt, err := LineCount(path, ext == "")
		if err != nil {
			return err
		}
		if ext == "" && scriptExt != "" {
			ext = scriptExt
		}
		w.mu.Lock()
		w.exts[ext] += lines
		w.mu.Unlock()
		return nil
	}
	if typ&os.ModeDir != 0 {
		if seen {
			return filepath.SkipDir
		}
		base := filepath.Base(path)
		if base == "" || base[0] == '.' || base[0] == '_' ||
			base == "testdata" || base == "node_modules" {
			return filepath.SkipDir
		}
		return nil
	}

	return nil
}

// CEV: awful name - fixme
type Count struct {
	S, L string
	N    int64
}

type byName []Count

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].L < b[j].L }

type byCount []Count

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].N < b[j].N }

type byNameCount []Count

func (b byNameCount) Len() int      { return len(b) }
func (b byNameCount) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byNameCount) Less(i, j int) bool {
	return b[i].N < b[j].N || (b[i].N == b[j].N && b[i].S < b[j].S)
}

type StringSliceValue []string

var _ flag.Getter = (*StringSliceValue)(nil)

func (v StringSliceValue) Get() interface{} {
	return ([]string)(v)
}

func (v *StringSliceValue) Set(s string) error {
	*v = append(*v, s)
	return nil
}

func (v StringSliceValue) String() string {
	return fmt.Sprintf("%q", ([]string)(v))
}

const ProgramName = "linecount"

var FollowSymlinks bool

func parseFlags() *flag.FlagSet {
	set := flag.NewFlagSet(ProgramName, flag.ExitOnError)

	set.BoolVar(&FollowSymlinks, "L", false, "Follow symlinks")

	set.Usage = func() {
		fmt.Fprintf(set.Output(), "%s: [OPTIONS] [PATH...]\n", set.Name())
		flag.PrintDefaults()
	}
	// error handled by flag.ExitOnError
	set.Parse(os.Args[1:])
	return set
}

func main() {
	var UseThousandsSeparators bool
	var IgnoredNames StringSliceValue
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: [OPTIONS] [PATH...]\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.BoolVar(&UseThousandsSeparators, "n", false,
		"Print numbers with thousands separators.")
	flag.Var(&IgnoredNames, "x", "Ignore directories.")
	flag.Parse()

	pwd, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, pwd)
	}

	w := Walker{
		exts: make(map[string]int64),
	}
	if len(IgnoredNames) != 0 {
		w.ignore = make(map[string]bool, len(IgnoredNames))
		for _, s := range IgnoredNames {
			w.ignore[s] = true
		}
	}

	for _, path := range args {
		if !filepath.IsAbs(path) {
			path = filepath.Join(pwd, path)
		}
		if !isDir(path) {
			fmt.Fprintf(os.Stderr, "%s: skipping not a directory\n", path)
			continue
		}
		if err := fastwalk.Walk(path, w.Walk); err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %s\n", path, err)
		}
	}

	var total int64
	exts := make([]Count, 0, len(w.exts))
	for s, n := range w.exts {
		if s == "" {
			s = "<none>"
		}
		exts = append(exts, Count{S: s, N: n})
		total += n
	}

	sort.Sort(byNameCount(exts))

	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, l := range exts {
		b = b[:0]
		if UseThousandsSeparators {
			b = append(b, num.FormatInt(l.N)...)
		} else {
			b = strconv.AppendInt(b, l.N, 10)
		}
		b = append(b, ':')
		b = append(b, '\t')
		b = append(b, l.S...)
		b = append(b, '\n')
		if _, err := wr.Write(b); err != nil {
			Fatal(err)
		}
	}
	// TODO: print total
	if err := wr.Flush(); err != nil {
		Fatal(err)
	}
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	errMsg := "Error"
	if _, file, line, _ := runtime.Caller(1); file != "" {
		errMsg = fmt.Sprintf("Error (%s:#%d)", filepath.Base(file), line)
	}
	switch e := err.(type) {
	case string, error, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", errMsg, e)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", errMsg, e)
	}
	os.Exit(1)
}

/*
func LineCount(r io.Reader, buf []byte) (lines int64, err error) {
	if buf == nil {
		buf = make([]byte, 32*1024)
	}
	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			lines += int64(bytes.Count(buf[0:nr], []byte{'\n'}))
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}
*/
