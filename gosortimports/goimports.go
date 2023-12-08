// Most of the code here is borrowed from golang.org/x/tools/cmd/goimports
//
// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

// TODO: add an "-auto" flag for guessing the local import path

var (
	// main operation modes
	list   = flag.Bool("l", false, "list files whose formatting differs from goimport's")
	write  = flag.Bool("w", false, "write result to (source) file instead of stdout")
	doDiff = flag.Bool("d", false, "display diffs instead of rewriting files")
	srcdir = flag.String("srcdir", "", "choose imports as if source code is from `dir`. "+
		"When operating on a single file, dir may instead be the complete file name.")
	simplifyAST = flag.Bool("s", false, "simplify code (same as `gofmt -s`")

	verbose bool // verbose logging

	cpuProfile     = flag.String("cpuprofile", "", "CPU profile output")
	memProfile     = flag.String("memprofile", "", "memory profile output")
	memProfileRate = flag.Int("memrate", 0, "if > 0, sets runtime.MemProfileRate")

	options = &imports.Options{
		TabWidth:  8,
		TabIndent: true,
		Comments:  true,
		Fragment:  true,
	}
	exitCode = int32(0)
)

func init() {
	flag.BoolVar(&options.AllErrors, "e", false,
		"report all errors (not just the first 10 on different lines)")
	flag.StringVar(&imports.LocalPrefix, "local", "",
		"put imports beginning with this string after 3rd-party packages; comma-separated list")
	flag.BoolVar(&options.FormatOnly, "format-only", false,
		"if true, don't fix imports and only format. In this mode, goimports is effectively gofmt, "+
			"with the addition that imports are grouped into sections.")
}

func setExitCode(code int32) {
	atomic.StoreInt32(&exitCode, code)
}

// TODO: print all errors on exit
func report(err error) {
	scanner.PrintError(os.Stderr, err)
	setExitCode(2)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [flags] [path ...]\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(2)
}

// argumentType is which mode goimports was invoked as.
type argumentType int

const (
	// fromStdin means the user is piping their source into goimports.
	fromStdin argumentType = iota

	// singleArg is the common case from editors, when goimports is run on
	// a single file.
	singleArg

	// multipleArg is when the user ran "goimports file1.go file2.go"
	// or ran goimports on a directory tree.
	multipleArg
)

func processFile(filename string, in io.Reader, out io.Writer, argType argumentType) error {
	opt := options
	if argType == fromStdin {
		nopt := *options
		nopt.Fragment = true
		opt = &nopt
	}

	if in == nil {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	src, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	src, err = mergeSortedImports(filename, src)
	if err != nil {
		return err
	}

	target := filename
	if *srcdir != "" {
		// Determine whether the provided -srcdirc is a directory or file
		// and then use it to override the target.
		//
		// See https://github.com/dominikh/go-mode.el/issues/146
		if isFile(*srcdir) {
			if argType == multipleArg {
				return errors.New("-srcdir value can't be a file when passing multiple arguments or when walking directories")
			}
			target = *srcdir
		} else if argType == singleArg && strings.HasSuffix(*srcdir, ".go") && !isDir(*srcdir) {
			// For a file which doesn't exist on disk yet, but might shortly.
			// e.g. user in editor opens $DIR/newfile.go and newfile.go doesn't yet exist on disk.
			// The goimports on-save hook writes the buffer to a temp file
			// first and runs goimports before the actual save to newfile.go.
			// The editor's buffer is named "newfile.go" so that is passed to goimports as:
			//      goimports -srcdir=/gopath/src/pkg/newfile.go /tmp/gofmtXXXXXXXX.go
			// and then the editor reloads the result from the tmp file and writes
			// it to newfile.go.
			target = *srcdir
		} else {
			// Pretend that file is from *srcdir in order to decide
			// visible imports correctly.
			target = filepath.Join(*srcdir, filepath.Base(filename))
		}
	}

	res, err := imports.Process(target, src, opt)
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		if *list {
			fmt.Fprintln(out, filename)
		}
		if *write {
			if argType == fromStdin {
				// filename is "<standard input>"
				return errors.New("can't use -w on stdin")
			}
			// On Windows, we need to re-set the permissions from the file. See golang/go#38225.
			var perms os.FileMode
			if fi, err := os.Stat(filename); err == nil {
				perms = fi.Mode() & os.ModePerm
			}
			err = os.WriteFile(filename, res, perms)
			if err != nil {
				return err
			}
		}
		if *doDiff {
			if argType == fromStdin {
				filename = "stdin.go" // because <standard input>.orig looks silly
			}
			data, err := diff(src, res, filename)
			if err != nil {
				return fmt.Errorf("computing diff: %s", err)
			}
			fmt.Printf("diff -u %s %s\n", filepath.ToSlash(filename+".orig"), filepath.ToSlash(filename))
			out.Write(data)
		}
	}

	if !*list && !*write && !*doDiff {
		_, err = out.Write(res)
	}

	return err
}

// parse parses src, which was read from filename,
// as a Go source file or statement list.
func parse(fset *token.FileSet, filename string, src []byte, opt *imports.Options) (*ast.File, error) {
	parserMode := parser.Mode(0)
	if opt.Comments {
		parserMode |= parser.ParseComments
	}
	if opt.AllErrors {
		parserMode |= parser.AllErrors
	}

	// Try as whole source file.
	file, err := parser.ParseFile(fset, filename, src, parserMode)
	if err == nil {
		return file, nil
	}
	// If the error is that the source file didn't begin with a
	// package line and we accept fragmented input, fall through to
	// try as a source fragment.  Stop and return on any other error.
	if !opt.Fragment || !strings.Contains(err.Error(), "expected 'package'") {
		return nil, err
	}

	// If this is a declaration list, make it a source file
	// by inserting a package clause.
	// Insert using a ;, not a newline, so that parse errors are on
	// the correct line.
	const prefix = "package main;"
	psrc := append([]byte(prefix), src...)
	file, err = parser.ParseFile(fset, filename, psrc, parserMode)
	if err == nil {
		// Gofmt will turn the ; into a \n.
		// Do that ourselves now and update the file contents,
		// so that positions and line numbers are correct going forward.
		psrc[len(prefix)-1] = '\n'
		fset.File(file.Package).SetLinesForContent(psrc)

		// If a main function exists, we will assume this is a main
		// package and leave the file.
		if containsMainFunc(file) {
			return file, nil
		}

		return file, nil
	}
	// If the error is that the source file didn't begin with a
	// declaration, fall through to try as a statement list.
	// Stop and return on any other error.
	if !strings.Contains(err.Error(), "expected declaration") {
		return nil, err
	}

	// If this is a statement list, make it a source file
	// by inserting a package clause and turning the list
	// into a function body.  This handles expressions too.
	// Insert using a ;, not a newline, so that the line numbers
	// in fsrc match the ones in src.
	fsrc := append(append([]byte("package p; func _() {"), src...), '}')
	file, err = parser.ParseFile(fset, filename, fsrc, parserMode)
	if err == nil {
		return file, nil
	}

	// Failed, and out of options.
	return nil, err
}

// containsMainFunc checks if a file contains a function declaration with the
// function signature 'func main()'
func containsMainFunc(file *ast.File) bool {
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok {
			if f.Name.Name != "main" {
				continue
			}

			if len(f.Type.Params.List) != 0 {
				continue
			}

			if f.Type.Results != nil && len(f.Type.Results.List) != 0 {
				continue
			}

			return true
		}
	}

	return false
}

func unquote(s string) string {
	if len(s) != 0 && s[0] == '`' || s[0] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// mergeSortedImports merges all imports into one sorted block ignoring grouping.
func mergeSortedImports(filename string, src []byte) ([]byte, error) {
	mode := parser.Mode(0)
	if options.Comments {
		mode |= parser.ParseComments
	}
	if options.AllErrors {
		mode |= parser.AllErrors
	}
	fset := token.NewFileSet()

	af, err := parse(fset, filename, src, options)
	if err != nil {
		return nil, err
	}

	imps := astutil.Imports(fset, af)
	for _, block := range imps {
		if len(block) == 1 && unquote(block[0].Path.Value) == "C" {
			continue
		}
		for _, m := range block {
			if m.Name != nil {
				astutil.DeleteNamedImport(fset, af, m.Name.Name, unquote(m.Path.Value))
			} else {
				astutil.DeleteImport(fset, af, unquote(m.Path.Value))
			}
		}
	}

	for _, block := range imps {
		if len(block) == 1 && unquote(block[0].Path.Value) == "C" {
			continue
		}
		for _, m := range block {
			if m.Name != nil {
				astutil.AddNamedImport(fset, af, m.Name.Name, unquote(m.Path.Value))
			} else {
				astutil.AddImport(fset, af, unquote(m.Path.Value))
			}
		}
	}

	if *simplifyAST {
		simplify(af)
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, af); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// call gofmtMain in a separate function
	// so that it can use defer and have them
	// run before the exit.
	gofmtMain()
	os.Exit(int(atomic.LoadInt32(&exitCode)))
}

// parseFlags parses command line flags and returns the paths to process.
// It's a var so that custom implementations can replace it in other files.
var parseFlags = func() []string {
	flag.BoolVar(&verbose, "v", false, "verbose logging")

	flag.Parse()
	return flag.Args()
}

func bufferedFileWriter(dest string) (w io.Writer, close func()) {
	f, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	bw := bufio.NewWriter(f)
	return bw, func() {
		if err := bw.Flush(); err != nil {
			log.Fatalf("error flushing %v: %v", dest, err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}

func gofmtMain() {
	flag.Usage = usage
	paths := parseFlags()

	if *cpuProfile != "" {
		bw, flush := bufferedFileWriter(*cpuProfile)
		pprof.StartCPUProfile(bw)
		defer flush()
		defer pprof.StopCPUProfile()
	}
	if *memProfileRate > 0 {
		runtime.MemProfileRate = *memProfileRate
		bw, flush := bufferedFileWriter(*memProfile)
		defer func() {
			runtime.GC() // materialize all statistics
			if err := pprof.WriteHeapProfile(bw); err != nil {
				log.Fatal(err)
			}
			flush()
		}()
	}

	if verbose {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		imports.Debug = true
	}
	if options.TabWidth < 0 {
		fmt.Fprintf(os.Stderr, "negative tabwidth %d\n", options.TabWidth)
		setExitCode(2)
		return
	}

	if len(paths) == 0 {
		if err := processFile("<standard input>", os.Stdin, os.Stdout, fromStdin); err != nil {
			report(err)
		}
		return
	}

	argType := singleArg
	if len(paths) > 1 {
		argType = multipleArg
	}

	// Quick check for a single file
	if len(paths) == 1 && strings.HasSuffix(paths[0], ".go") && isFile(paths[0]) {
		if err := processFile(paths[0], nil, os.Stdout, argType); err != nil {
			report(err)
		}
		return
	}

	type request struct {
		filename string
		argType  argumentType
	}

	numWorkers := runtime.NumCPU()
	switch {
	case numWorkers < 1:
		numWorkers = 4
	case numWorkers >= 32:
		numWorkers = 32
	}
	// Don't spin up more workers than we have arguments
	if !hasDir(paths) && len(paths) < numWorkers {
		numWorkers = len(paths)
	}

	ch := make(chan *request, numWorkers*4)

	visitFile := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			report(err)
			return nil
		}
		name := d.Name()
		if !d.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") {
			ch <- &request{filename: path, argType: multipleArg}
		}
		return nil
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range ch {
				if err := processFile(r.filename, nil, os.Stdout, r.argType); err != nil {
					report(err)
				}
			}
		}()
	}

	for _, path := range paths {
		switch dir, err := os.Stat(path); {
		case err != nil:
			report(err)
		case dir.IsDir():
			// walkDir(path)
			filepath.WalkDir(path, visitFile)
		default:
			// if err := processFile(path, nil, os.Stdout, argType); err != nil {
			// 	report(err)
			// }
			ch <- &request{filename: path, argType: argType}
		}
	}
	close(ch)
	wg.Wait()
}

func writeTempFile(dir, prefix string, data []byte) (string, error) {
	file, err := os.CreateTemp(dir, prefix)
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func diff(b1, b2 []byte, filename string) (data []byte, err error) {
	f1, err := writeTempFile("", "gofmt", b1)
	if err != nil {
		return
	}
	defer os.Remove(f1)

	f2, err := writeTempFile("", "gofmt", b2)
	if err != nil {
		return
	}
	defer os.Remove(f2)

	cmd := "diff"
	if runtime.GOOS == "plan9" {
		cmd = "/bin/ape/diff"
	}

	data, err = exec.Command(cmd, "-u", f1, f2).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		return replaceTempFilename(data, filename)
	}
	return
}

// replaceTempFilename replaces temporary filenames in diff with actual one.
//
// --- /tmp/gofmt316145376	2017-02-03 19:13:00.280468375 -0500
// +++ /tmp/gofmt617882815	2017-02-03 19:13:00.280468375 -0500
// ...
// ->
// --- path/to/file.go.orig	2017-02-03 19:13:00.280468375 -0500
// +++ path/to/file.go	2017-02-03 19:13:00.280468375 -0500
// ...
func replaceTempFilename(diff []byte, filename string) ([]byte, error) {
	bs := bytes.SplitN(diff, []byte{'\n'}, 3)
	if len(bs) < 3 {
		return nil, fmt.Errorf("got unexpected diff for %s", filename)
	}
	// Preserve timestamps.
	var t0, t1 []byte
	if i := bytes.LastIndexByte(bs[0], '\t'); i != -1 {
		t0 = bs[0][i:]
	}
	if i := bytes.LastIndexByte(bs[1], '\t'); i != -1 {
		t1 = bs[1][i:]
	}
	// Always print filepath with slash separator.
	f := filepath.ToSlash(filename)
	bs[0] = []byte(fmt.Sprintf("--- %s%s", f+".orig", t0))
	bs[1] = []byte(fmt.Sprintf("+++ %s%s", f, t1))
	return bytes.Join(bs, []byte{'\n'}), nil
}

// isFile reports whether name is a file.
func isFile(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.Mode().IsRegular()
}

// isDir reports whether name is a directory.
func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

// hasDir reports if one of paths is a directory and not a Go file
func hasDir(paths []string) bool {
	for _, path := range paths {
		if !strings.HasSuffix(path, ".go") && isDir(path) {
			return true
		}
	}
	return false
}
