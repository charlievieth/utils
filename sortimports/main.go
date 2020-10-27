package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

var (
	zeroDelim  = false
	formatOnly = false

	options = &imports.Options{
		Fragment:  true,
		AllErrors: true,
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	}
	exitCode = 0
)

func init() {
	flag.StringVar(&imports.LocalPrefix, "local", "",
		"put imports beginning with this string after 3rd-party packages; "+
			"comma-separated list")

	flag.BoolVar(&formatOnly, "sort-only", false,
		"if true, don't fix imports and only format and sort imports")

	flag.BoolVar(&zeroDelim, "0", false,
		"expect NUL ('\\0') characters as separators, instead of newlines "+
			"when reading files from STDIN")
}

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [flags] [path ...]\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(2)
}

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func unquote(s string) string {
	if len(s) != 0 && s[0] == '`' || s[0] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func processFile(filename string) error {
	fset := token.NewFileSet()

	fi, err := os.Lstat(filename)
	if err != nil {
		return err
	}
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	af, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
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
		if len(block) == 1 && block[0].Path.Value == `"C"` {
			continue
		}
		for _, m := range block {
			if m.Name != nil {
				path, _ := strconv.Unquote(m.Path.Value)
				astutil.AddNamedImport(fset, af, m.Name.Name, path)
			} else {
				path, _ := strconv.Unquote(m.Path.Value)
				astutil.AddImport(fset, af, path)
			}
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, af); err != nil {
		return err
	}
	if !bytes.Equal(buf.Bytes(), src) {
		if err := ioutil.WriteFile(filename, buf.Bytes(), fi.Mode()); err != nil {
			return err
		}
	}

	res, err := imports.Process(filename, buf.Bytes(), options)
	if err != nil {
		return err
	}
	if bytes.Equal(src, res) {
		return nil
	}

	return ioutil.WriteFile(filename, res, fi.Mode())
}

func visitFile(path string, f os.FileInfo, err error) error {
	if err == nil && isGoFile(f) {
		err = processFile(path)
	}
	if err != nil {
		report(err)
	}
	return nil
}

func walkDir(path string) {
	filepath.Walk(path, visitFile)
}

func readFilesFromStdin() ([]string, error) {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	var delim byte
	if !zeroDelim {
		delim = '\n'
	}

	a := bytes.Split(in, []byte{delim})

	paths := make([]string, 0, len(a))
	for _, b := range a {
		s := string(b)
		if strings.TrimSpace(s) == "" {
			continue
		}
		paths = append(paths, s)
	}
	return paths, nil
}

func realMain() {
	flag.Usage = usage
	flag.Parse()

	paths := flag.Args()

	// read files from stdin
	if len(paths) == 0 || paths[0] == "-" {
		var err error
		paths, err = readFilesFromStdin()
		if err != nil {
			report(err)
		}
		return
	}

	for _, path := range flag.Args() {
		switch dir, err := os.Stat(path); {
		case err != nil:
			report(err)
		case dir.IsDir():
			walkDir(path)
		default:
			if err := processFile(path); err != nil {
				report(err)
			}
		}
	}
}

func main() {
	realMain()
	os.Exit(exitCode)
}
