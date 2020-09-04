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

func WellKnownFilename(s string) bool {
	switch s {
	case "Dockerfile", "Gemfile", "Makefile", "Podfile", "Rakefile",
		"CMakeLists.txt", "LICENSE", "MANIFEST", "METADATA", "NOTICE",
		"AUTHORS", "CODEOWNERS", "CONTRIBUTORS", "README", "PATENTS",
		"OWNERS", "BUILD", "WORKSPACE":
		return true
	}
	return false
}

func IgnoredExtension(ext string) bool {
	switch ext {
	case ".bz", ".bzip", ".exe", ".gz", ".gzip", ".tar", ".tbz", ".tgz",
		".vdi", ".xz", ".zip":
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

func LineCount(filename string) (int64, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	p := bufPool.Get().(*[]byte)
	defer func() {
		f.Close()
		bufPool.Put(p)
	}()
	buf := *p

	nr, err := f.Read(buf)
	if isBinary(buf[0:nr]) {
		return 0, ErrBinary
	}
	lines := int64(bytes.Count(buf[0:nr], newLine))
	if err != nil {
		if err != io.EOF {
			return 0, err
		}
		return lines, nil
	}

	for {
		nr, er := f.Read(buf)
		lines += int64(bytes.Count(buf[0:nr], newLine))
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return lines, err
}

type Walker struct {
	exts     map[string]int64
	ignore   map[string]bool
	mu       sync.Mutex
	seen     *SeenFiles
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
		lines, err := LineCount(path)
		if err != nil {
			if err != ErrBinary {
				return err
			}
			return nil
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
		if w.ignore != nil && w.ignore[base] {
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
	seen := w.seen.Seen(path)
	if typ.IsRegular() {
		if seen {
			return nil
		}
		lines, err := LineCount(path)
		if err != nil {
			return err
		}
		ext := Ext(path)
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
