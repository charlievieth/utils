package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	if len(os.Args) != 2 {
		Fatal("USAGE")
	}
	filename := os.Args[1]
	_ = filename

	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		Fatal(err)
	}

	fmt.Println(len(af.Imports))
	for i, m := range af.Imports {
		if m.Name != nil {
			fmt.Printf("%d: %s\n", i, m.Name.Name)
		}
		if m.Path != nil {
			fmt.Printf("%d: %s\n", i, m.Path.Value)
		}
	}
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
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
