package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/charlievieth/fastwalk"
	"github.com/charlievieth/num"
	"github.com/spf13/cobra"
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

func wellKnownFilename(s string) bool {
	switch s {
	case "Dockerfile", "Gemfile", "Makefile", "Podfile", "Rakefile",
		"CMakeLists.txt", "LICENSE", "MANIFEST", "METADATA", "NOTICE",
		"AUTHORS", "CODEOWNERS", "CONTRIBUTORS", "README", "PATENTS",
		"OWNERS", "BUILD", "WORKSPACE", "tags":
		return true
	}
	return false
}

func ignoredExtension(ext string) bool {
	switch ext {
	case ".a", ".bz", ".bzip", ".exe", ".gz", ".gzip", ".la", ".so", ".tar",
		".tbz", ".tgz", ".vdi", ".xz", ".zip", ".zst":
		return true
	}
	return false
}

func normalizeExt(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case "":
		base := filepath.Base(path)
		if wellKnownFilename(base) {
			ext = base
		}
	case ".txt":
		if strings.HasSuffix(path, "CMakeLists.txt") {
			ext = "CMakeLists.txt"
		}
	}
	return ext
}

func executableMode(m os.FileMode) bool {
	const mask = 1 | 8 | 64
	return m&mask != 0
}

// Tested with multiple sizes of 8k and 96k seems best.
// Smaller sizes tended to under perform compared to mmap,
// which was slower for almost all sizes when 96k was used.
const bufSize = 96 * 1024

var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, bufSize)
		return &b
	},
}

const maxBinaryReadSize = 256

var ErrBinary = errors.New("binary file")

func isBinary(b []byte) bool {
	if len(b) >= maxBinaryReadSize {
		b = b[:maxBinaryReadSize]
	}
	return bytes.IndexByte(b, 0) != -1
}

var newLine = []byte{'\n'}

func lineCountFile(f *os.File, needExt bool) (lines int64, ext string, err error) {
	p := bufPool.Get().(*[]byte)
	defer bufPool.Put(p)
	buf := *p

	nr, err := f.Read(buf)
	if isBinary(buf[0:nr]) {
		return 0, "", ErrBinary
	}
	if needExt {
		ext = extractShebang(buf[0:nr])
	}
	for {
		lines += int64(bytes.Count(buf[0:nr], newLine))
		if err != nil {
			break
		}
		nr, err = f.Read(buf)
	}
	if err != nil && err == io.EOF {
		err = nil
	}
	return lines, ext, err
}

func lineCount(filename string, needExt bool) (int64, string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	return lineCountFile(f, needExt)
}

type walker struct {
	mu     sync.Mutex
	exts   map[string]int64
	ignore map[string]bool
}

func (w *walker) Walk(path string, de fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	typ := de.Type()
	if typ.IsRegular() {
		ext := normalizeExt(path)
		if ignoredExtension(ext) {
			return nil
		}
		lines, scriptExt, err := lineCount(path, ext == "")
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

// CEV: awful name - fixme
type extLineCount struct {
	Lines int64
	Ext   string
	Lower string
}

type byNameCount []extLineCount

func (b byNameCount) Len() int      { return len(b) }
func (b byNameCount) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byNameCount) Less(i, j int) bool {
	b1 := b[i]
	b2 := b[j]
	return b1.Lines < b2.Lines || (b1.Lines == b2.Lines && b1.Ext < b2.Ext)
}

type byNameCountIgnoreCase []extLineCount

func (b byNameCountIgnoreCase) Len() int      { return len(b) }
func (b byNameCountIgnoreCase) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byNameCountIgnoreCase) Less(i, j int) bool {
	b1 := b[i]
	b2 := b[j]
	return b1.Lines < b2.Lines || (b1.Lines == b2.Lines && b1.Lower < b2.Lower)
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func generateShellCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("invalid shell: %q", shell)
	}
}

func realMain() error {
	root := cobra.Command{
		Use: "fastwalk: [OPTIONS] [PATH...]",
	}
	flags := root.Flags()
	flags.SortFlags = false

	// TODO: support `rg` style globs
	//
	// flags.StringArrayP("glob", "g", nil, "Ignore directories matching GLOB.")

	flags.StringArrayP("exclude", "x", nil,
		"Ignore directories matching GLOB.")
	flags.BoolP("pretty-numbers", "n", false,
		"Print numbers with thousands separators.")
	flags.BoolP("ignore-case", "s", false,
		"Ignore case when sorting file extensions.")

	// TODO: add an option to ignore duplicate files (expensive)
	flags.BoolP("follow", "L", false,
		"Follow symbolic links while traversing directories.")

	flags.String("completion", "", "Generate base completion for SHELL")

	cpuprofile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flags.String("memprofile", "", "write memory profile to `file`")

	var defuncs []func()
	defer func() {
		for i := len(defuncs) - 1; i >= 0; i-- {
			defuncs[i]()
		}
	}()
	atexit := func(fn func()) { defuncs = append(defuncs, fn) }

	root.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				return fmt.Errorf("could not create CPU profile: %w", err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				_ = f.Close()
				return fmt.Errorf("could not start CPU profile: %w", err)
			}
			atexit(func() {
				pprof.StopCPUProfile()
				_ = f.Close()
			})
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				return fmt.Errorf("could not create memory profile: %w", err)
			}
			atexit(func() {
				runtime.GC() // get up-to-date statistics
				err := pprof.WriteHeapProfile(f)
				_ = f.Close()
				if err != nil {
					panic(fmt.Sprint("could not write memory profile:", err))
				}
			})
		}
		return nil
	}

	root.RunE = func(cmd *cobra.Command, args []string) error {
		// Generate shell completion and exit
		if shell, err := cmd.Flags().GetString("completion"); err == nil && shell != "" {
			return generateShellCompletion(cmd, shell)
		}

		exclude, err := cmd.Flags().GetStringArray("exclude")
		if err != nil {
			return err
		}
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			return err
		}
		prettyNumbers, err := cmd.Flags().GetBool("pretty-numbers")
		if err != nil {
			return err
		}
		ignoreCase, err := cmd.Flags().GetBool("ignore-case")
		if err != nil {
			return err
		}

		if len(args) == 0 {
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}
			args = []string{pwd}
		}

		w := &walker{
			exts: make(map[string]int64),
		}
		if len(exclude) != 0 {
			w.ignore = make(map[string]bool, len(exclude))
			for _, s := range exclude {
				w.ignore[s] = true
			}
		}
		conf := fastwalk.Config{
			Follow: follow,
		}

		// TODO: follow symlinks provided on the command line?
		for _, path := range args {
			if !isDir(path) {
				fmt.Fprintf(os.Stderr, "%s: skipping not a directory\n", path)
				continue
			}
			if err := fastwalk.Walk(&conf, path, w.Walk); err != nil {
				fmt.Fprintf(os.Stderr, "%s: error: %s\n", path, err)
			}
		}

		var total int64
		exts := make([]extLineCount, 0, len(w.exts))
		for s, n := range w.exts {
			if s == "" {
				s = "<none>"
			}
			exts = append(exts, extLineCount{
				Lines: n,
				Ext:   s,
			})
			total += n
		}

		if ignoreCase {
			for i, e := range exts {
				exts[i].Lower = strings.ToLower(e.Ext)
			}
			sort.Sort(byNameCountIgnoreCase(exts))
		} else {
			sort.Sort(byNameCount(exts))
		}

		wr := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		b := make([]byte, 0, 128)
		for _, l := range exts {
			b = b[:0]
			if prettyNumbers {
				b = append(b, num.FormatInt(l.Lines)...)
			} else {
				b = strconv.AppendInt(b, l.Lines, 10)
			}
			b = append(b, ':')
			b = append(b, '\t')
			b = append(b, l.Ext...)
			b = append(b, '\n')
			if _, err := wr.Write(b); err != nil {
				return err
			}
		}
		// TODO: print total
		if err := wr.Flush(); err != nil {
			return err
		}
		return nil
	}

	return root.Execute()
}

func main() {
	if err := realMain(); err != nil {
		os.Exit(1)
	}
}
