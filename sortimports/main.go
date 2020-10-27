package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
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
	// CEV: I'm too lazy to figure out all the logic for merging empty lines
	// but running it twice works - so we run it three times just to be sure.
	for i := 0; i < 5; i++ {
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
			i := 0
			specs := d.Specs[:0]
			for j, s := range d.Specs {
				if j > i && fset.Position(s.Pos()).Line > 1+fset.Position(d.Specs[j-1].End()).Line {
					// j begins a new run. End this one.
					specs = append(specs, d.Specs[i:j]...)
					fset.File(d.Specs[j-1].End()).MergeLine(fset.Position(d.Specs[j-1].End()).Line)
					i = j
				}
			}
			specs = append(specs, d.Specs[i:]...)

			// TODO: try to change the position of the specs
			//
			// fset.File(1).SetLines(lines)

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
}

func FixFile(name string) error {
	fset := token.NewFileSet()

	af, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
	if err != nil {
		Fatal(err)
	}
	SortImports(fset, af)

	var out bytes.Buffer
	if err := printer.Fprint(&out, fset, af); err != nil {
		return err
	}

	fi, err := os.Lstat(name)
	if err != nil {
		return err
	}
	tmpname := name + ".tmp"
	f, err := os.OpenFile(tmpname, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fi.Mode())
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := out.WriteTo(f); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmpname, name)
}

func main() {
	localPkg := flag.String("local", "", "put imports beginning with this string after 3rd-party packages; comma-separated list")
	flag.Parse()
	_ = localPkg

	if flag.NArg() == 0 {
		return
	}

	var failed bool
	for _, filename := range flag.Args() {
		if err := FixFile(filename); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %s\n", filename, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}

	args := []string{"-w"}
	if *localPkg != "" {
		args = append(args, "-local", *localPkg)
	}
	args = append(args, flag.Args()...)

	cmd := exec.Command("goimports", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
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

/*

// collapse indicates whether prev may be removed, leaving only next.
func collapse(prev, next ast.Spec) bool {
	if importPath(next) != importPath(prev) || importName(next) != importName(prev) {
		return false
	}
	return prev.(*ast.ImportSpec).Comment == nil
}

type posSpan struct {
	Start token.Pos
	End   token.Pos
}

func sortSpecs(fset *token.FileSet, f *ast.File, specs []ast.Spec) []ast.Spec {
	// Can't short-circuit here even if specs are already sorted,
	// since they might yet need deduplication.
	// A lone import, however, may be safely ignored.
	if len(specs) <= 1 {
		return specs
	}

	// Record positions for specs.
	pos := make([]posSpan, len(specs))
	for i, s := range specs {
		pos[i] = posSpan{s.Pos(), s.End()}
	}

	// Identify comments in this range.
	// Any comment from pos[0].Start to the final line counts.
	lastLine := fset.Position(pos[len(pos)-1].End).Line
	cstart := len(f.Comments)
	cend := len(f.Comments)
	for i, g := range f.Comments {
		if g.Pos() < pos[0].Start {
			continue
		}
		if i < cstart {
			cstart = i
		}
		if fset.Position(g.End()).Line > lastLine {
			cend = i
			break
		}
	}
	comments := f.Comments[cstart:cend]

	// Assign each comment to the import spec preceding it.
	importComment := map[*ast.ImportSpec][]*ast.CommentGroup{}
	specIndex := 0
	for _, g := range comments {
		for specIndex+1 < len(specs) && pos[specIndex+1].Start <= g.Pos() {
			specIndex++
		}
		s := specs[specIndex].(*ast.ImportSpec)
		importComment[s] = append(importComment[s], g)
	}

	// Sort the import specs by import path.
	// Remove duplicates, when possible without data loss.
	// Reassign the import paths to have the same position sequence.
	// Reassign each comment to abut the end of its spec.
	// Sort the comments by new position.
	sort.Sort(byImportSpec(specs))

	// Dedup. Thanks to our sorting, we can just consider
	// adjacent pairs of imports.
	deduped := specs[:0]
	for i, s := range specs {
		if i == len(specs)-1 || !collapse(s, specs[i+1]) {
			deduped = append(deduped, s)
		} else {
			p := s.Pos()
			fset.File(p).MergeLine(fset.Position(p).Line)
		}
	}
	specs = deduped

	// Fix up comment positions
	for i, s := range specs {
		s := s.(*ast.ImportSpec)
		if s.Name != nil {
			s.Name.NamePos = pos[i].Start
		}
		s.Path.ValuePos = pos[i].Start
		s.EndPos = pos[i].End
		nextSpecPos := pos[i].End

		for _, g := range importComment[s] {
			for _, c := range g.List {
				c.Slash = pos[i].End
				nextSpecPos = c.End()
			}
		}
		if i < len(specs)-1 {
			pos[i+1].Start = nextSpecPos
			pos[i+1].End = nextSpecPos
		}
	}

	sort.Sort(byCommentPos(comments))

	// Fixup comments can insert blank lines, because import specs are on different lines.
	// We remove those blank lines here by merging import spec to the first import spec line.
	firstSpecLine := fset.Position(specs[0].Pos()).Line
	for _, s := range specs[1:] {
		p := s.Pos()
		line := fset.File(p).Line(p)
		for previousLine := line - 1; previousLine >= firstSpecLine; {
			fset.File(p).MergeLine(previousLine)
			previousLine--
		}
	}
	return specs
}

type byImportSpec []ast.Spec // slice of *ast.ImportSpec

func (x byImportSpec) Len() int      { return len(x) }
func (x byImportSpec) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byImportSpec) Less(i, j int) bool {
	ipath := importPath(x[i])
	jpath := importPath(x[j])

	igroup := importGroup(ipath)
	jgroup := importGroup(jpath)
	if igroup != jgroup {
		return igroup < jgroup
	}

	if ipath != jpath {
		return ipath < jpath
	}
	iname := importName(x[i])
	jname := importName(x[j])

	if iname != jname {
		return iname < jname
	}
	return importComment(x[i]) < importComment(x[j])
}

type byCommentPos []*ast.CommentGroup

func (x byCommentPos) Len() int           { return len(x) }
func (x byCommentPos) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x byCommentPos) Less(i, j int) bool { return x[i].Pos() < x[j].Pos() }

// LocalPrefix is a comma-separated string of import path prefixes, which, if
// set, instructs Process to sort the import paths with the given prefixes
// into another group after 3rd-party packages.
const LocalPrefix = ""

func localPrefixes() []string {
	if LocalPrefix != "" {
		return strings.Split(LocalPrefix, ",")
	}
	return nil
}

// importToGroup is a list of functions which map from an import path to
// a group number.
var importToGroup = []func(importPath string) (num int, ok bool){
	func(importPath string) (num int, ok bool) {
		for _, p := range localPrefixes() {
			if strings.HasPrefix(importPath, p) || strings.TrimSuffix(p, "/") == importPath {
				return 3, true
			}
		}
		return
	},
	func(importPath string) (num int, ok bool) {
		if strings.HasPrefix(importPath, "appengine") {
			return 2, true
		}
		return
	},
	func(importPath string) (num int, ok bool) {
		if strings.Contains(importPath, ".") {
			return 1, true
		}
		return
	},
}

func importGroup(importPath string) int {
	for _, fn := range importToGroup {
		if n, ok := fn(importPath); ok {
			return n
		}
	}
	return 0
}
*/
