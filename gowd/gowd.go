package main

import (
	"fmt"
	"go/build"
	"os"
)

func realMain() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	pkg, err := build.ImportDir(wd, build.FindOnly)
	if err != nil {
		return "", err
	}
	return pkg.ImportPath, nil
}

func main() {
	gowd, err := realMain()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Println(gowd)
}
