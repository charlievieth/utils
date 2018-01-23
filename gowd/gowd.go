package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	pkg, err := build.ImportDir(wd, build.FindOnly)
	if err != nil {
		Fatal(err)
	}
	fmt.Println(pkg.ImportPath)
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
