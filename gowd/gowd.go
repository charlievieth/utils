package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"strings"
)

func tryGoList(wd string) (string, error) {
	data, err := exec.Command("go", "list", "-f", "{{.ImportPath}}", wd).CombinedOutput()
	out := strings.TrimSpace(string(data))
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return out, nil
}

func realMain() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	pkg, err := build.ImportDir(wd, 0)
	if err != nil {
		return "", err
	}
	if pkg.ImportPath != "." {
		return pkg.ImportPath, nil
	}
	return tryGoList(wd)
}

func main() {
	flag.Usage = func() {
		const msg = "gowd: print the import path of the Go package in the current directory.\n"
		fmt.Fprint(flag.CommandLine.Output(), msg)
		flag.PrintDefaults()
	}
	flag.Parse()

	gowd, err := realMain()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Println(gowd)
}
