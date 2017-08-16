// FUNCTION SEARCH
// FUNC SEARCH
// AST FUNC SEARCH

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"os"
	"sort"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	_ = sort.IntSlice([]int{1})
	_ = unicode.IsSpace(' ')
	_ = utf8.RuneError
	_ = fmt.Sprint("")
)

func Error(err error) {
	fmt.Println("Error:", err)
	os.Exit(1)
}

func PrintJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		Error(err)
	}
	return string(b)
}

func BenchScan(src []byte) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, scanner.ScanComments)

	t := time.Now()
	for {
		_, tok, lit := s.Scan()
		_ = lit
		if tok == token.EOF {
			break
		}
	}
	fmt.Println("Scan:", time.Since(t))
}

func BenchParse(src []byte) {
	fset := token.NewFileSet()
	t := time.Now()
	if _, err := parser.ParseFile(fset, "", src, parser.ParseComments); err != nil {
		Error(err)
	}
	fmt.Println("Parse:", time.Since(t))
}

func BenchSearch(src []byte) {
	t := time.Now()
	_ = bytes.Contains(src, []byte("typedslicecopy"))
	fmt.Println("Search:", time.Since(t))
}

func ContainsFunc(src, name []byte) bool {
	// i := 0
	// t := s[:len(s)-n+1]
	i := 0
	t := src[:len(src)-len(name)+1]
	for i < len(t) {
		o := bytes.Index(t[i:], []byte("func"))
		if o < 0 {
			break
		}
		n := bytes.LastIndex(t[i:i+o], []byte{'\n'})
		if n < 0 {
			break
		}
		n++
		fmt.Println(string(t[i+n : i+n+20]))
		// if !bytes.Contains(t[i:i+n], []byte("//")) {
		// 	fmt.Println(string(t[i+o : i+o+10]))
		// }
		i += o
		i++
	}
	return false
}

func MatchFunc(src, name []byte) bool {
	b := src
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		_ = r
		b = b[size:]
	}
	return false
}

func IsComment(s []byte) bool {
	i := 0
	for i < len(s) {
		if !IsWhitespace(s[i]) {
			break
		}
		i++
	}
	return bytes.HasPrefix(s[i:], []byte("//"))
}

func IsWhitespace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func StripCR(b []byte) []byte {
	if bytes.IndexByte(b, '\r') < 0 {
		return b
	}
	n := 0
	for i := 0; i < len(b); i++ {
		if b[i] != '\r' {
			b[n] = b[i]
			n++
		}
	}
	return b[:n]
}

const (
	stateInSource = iota
	stateInSlash
	stateInLine
	stateInBlock
	stateInStar
)

// For testing use ' ' for comment blocks to preserve Pos.
func TrimComments(b []byte) []byte {
	n := 0
	state := stateInSource
	for _, c := range b {
		switch state {
		case stateInSource:
			if c == '/' {
				state = stateInSlash
			}
			b[n] = c
			n++
		case stateInSlash:
			switch c {
			case '/':
				state = stateInLine
				n-- // remove last '/'
			case '*':
				state = stateInBlock
				n-- // remove last '/'
			default:
				state = stateInSource
			}
		case stateInLine:
			if c == '\n' {
				state = stateInSource
				// Preserve newlines
				b[n] = c
				n++
			}
		case stateInBlock:
			if c == '*' {
				state = stateInStar
			}
		case stateInStar:
			if c == '/' {
				state = stateInSource
			}
		}
	}
	return b[:n]
}

func Comments(b []byte) []Block {
	var blocks []Block
	pos := 0
	state := stateInSource
	for i, c := range b {
		switch state {
		case stateInSource:
			if c == '/' {
				state = stateInSlash
				pos = i
			}
		case stateInSlash:
			switch c {
			case '/':
				state = stateInLine
			case '*':
				state = stateInBlock
			default:
				state = stateInSource
			}
		case stateInLine:
			if c == '\n' {
				state = stateInSource
				blocks = append(blocks, Block{pos, i})
			}
		case stateInBlock:
			if c == '*' {
				state = stateInStar
			}
		case stateInStar:
			if c == '/' {
				state = stateInSource
				blocks = append(blocks, Block{pos, i})
			}
		}
	}
	return blocks
}

/* */

func CommentBlocks(p []byte) []Block {
	var b []Block
	i := 0
	t := p[:len(p)-1]
	for i < len(t) {
		o := bytes.IndexByte(t[i:], '/')
		if o < 0 {
			break
		}
		i += o
		switch t[i+1] {
		case '/':
			n := bytes.IndexByte(t[i:], '\n')
			if n < 0 {
				n = len(t) - i - 1 // set to EOF
			}
			b = append(b, Block{Pos: i, End: i + n})
			i += n
		case '*':
			n := bytes.Index(t[i:], []byte("*/"))
			if n < 0 {
				n = len(t) - i - 1 // set to EOF
			}
			b = append(b, Block{Pos: i, End: i + n})
			i += n
		}
		i++
	}
	return b
}

type BlockSlice []Block

func (b BlockSlice) Len() int           { return len(b) }
func (b BlockSlice) Less(i, j int) bool { return b[i].Pos < b[j].Pos }
func (b BlockSlice) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func (b BlockSlice) Search(pos int) int {
	i, j := 0, b.Len()
	for i < j {
		h := i + (j-i)/2
		if b[h].End < pos {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

func (b BlockSlice) In(pos int) bool {
	i := b.Search(pos)
	return i < len(b) && b[i].Pos <= pos && pos <= b[i].End
}

func InBlock(b []Block, pos int) bool {
	// if i := BlockSlice(b).Search(pos); i < len(b) {
	// 	return b[i].Pos <= pos && pos <= b[i].End
	// }
	return false
}

type Block struct {
	Pos, End int
}

func main() {
	// s := []byte("  // func equalPortable(a, b []byte) bool")
	// s := []byte("func equalPortable(a, b []byte) bool // comment")
	// fmt.Println(IsComment(s))
	// return

	src, err := ioutil.ReadFile("test/main.go")
	// src, err := ioutil.ReadFile("test/large.go")
	if err != nil {
		Error(err)
	}

	t := time.Now()
	b := BlockSlice(Comments(src))
	_ = b
	fmt.Println(time.Since(t))

	t = time.Now()
	c := BlockSlice(CommentBlocks(src))
	_ = c
	fmt.Println(time.Since(t))
	fmt.Println(len(b), len(c))

	// fmt.Printf("%d: %v\n", 1, b.In(1))
	// fmt.Printf("%d: %v\n", 120, b.In(120))
	// fmt.Printf("%d: %v\n", 200, b.In(200))

	// for i := 0; i < 20; i++ {
	// 	fmt.Printf("%+v\n", b[i])
	// }
}

func RunBench() {
	src, err := ioutil.ReadFile("test/main.go")
	if err != nil {
		Error(err)
	}
	BenchScan(src)
	BenchParse(src)
	BenchSearch(src)
	return
	b, err := ioutil.ReadFile("test/main.go")
	if err != nil {
		Error(err)
	}
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(b))
	var s scanner.Scanner
	s.Init(file, b, nil, scanner.ScanComments)

	inFunc := false
	var decl []string
	for {
		_, tok, lit := s.Scan()
		_ = lit
		if tok == token.EOF {
			break
		}
		if tok == token.FUNC && !inFunc {
			inFunc = true
		}
		if inFunc {
			if tok == token.LBRACE {
				inFunc = false
				fmt.Println(decl)
				decl = decl[:0]
			} else {
				decl = append(decl, fmt.Sprintf("%s:%s", lit, tok))
			}
		}
	}
}
