package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
)

/*func SortImports(filename string) error {
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	ast.SortImports(fset, af)

	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, af); err != nil {
		return err
	}

	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := out.WriteTo(f); err != nil {
		return err
	}
	return nil
}
*/

func importPath(s ast.Spec) string {
	t, err := strconv.Unquote(s.(*ast.ImportSpec).Path.Value)
	if err == nil {
		return t
	}
	return ""
}

func importName(s ast.Spec) string {
	n := s.(*ast.ImportSpec).Name
	if n == nil {
		return ""
	}
	return n.Name
}

func importComment(s ast.Spec) string {
	c := s.(*ast.ImportSpec).Comment
	if c == nil {
		return ""
	}
	return c.Text()
}

func SortImports(fset *token.FileSet, f *ast.File) {
	for _, d := range f.Decls {
		d, ok := d.(*ast.GenDecl)
		if !ok || d.Tok != token.IMPORT {
			// Not an import declaration, so we're done.
			// Imports are always first.
			break
		}

		if !d.Lparen.IsValid() {
			// Not a block: sorted by default.
			continue
		}

		// Identify and sort runs of specs on successive lines.
		// i := 0
		// for j, s := range d.Specs {
		// 	if j > i && fset.Position(s.Pos()).Line > 1+fset.Position(d.Specs[j-1].End()).Line {
		// 		// j begins a new run. End this one.
		// 		specs = append(specs, sortSpecs(fset, f, d.Specs[i:j])...)
		// 		i = j
		// 	}
		// }
		// specs = append(specs, sortSpecs(fset, f, d.Specs[i:])...)

		specs := make([]ast.Spec, len(d.Specs))
		copy(specs, d.Specs)

		for i, s := range specs {
			fmt.Printf("%d: %d::%d\n", i, s.Pos(), s.End())
		}

		sort.Slice(specs, func(i, j int) bool {
			ipath := importPath(specs[i])
			jpath := importPath(specs[j])
			if ipath != jpath {
				return ipath < jpath
			}
			iname := importName(specs[i])
			jname := importName(specs[j])
			if iname != jname {
				return iname < jname
			}
			return importComment(specs[i]) < importComment(specs[j])
		})
		d.Specs = specs

		// Deduping can leave a blank line before the rparen; clean that up.
		if len(d.Specs) > 0 {
			lastSpec := d.Specs[len(d.Specs)-1]
			lastLine := fset.Position(lastSpec.Pos()).Line
			rParenLine := fset.Position(d.Rparen).Line
			for rParenLine > lastLine+1 {
				rParenLine--
				fset.File(d.Rparen).MergeLine(rParenLine)
			}
		}
	}
}

func main() {
	filename := os.Args[1]
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		Fatal(err)
	}
	SortImports(fset, af)

	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, af); err != nil {
		Fatal(err)
	}

	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		Fatal(err)
	}
	defer f.Close()
	if _, err := out.WriteTo(f); err != nil {
		Fatal(err)
	}
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var s string

	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		s = "Error"
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
	}
	os.Exit(1)
}
