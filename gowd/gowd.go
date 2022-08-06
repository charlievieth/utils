package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func tryGoBuild(wd string) (string, error) {
	pkg, err := build.ImportDir(wd, 0)
	if err != nil {
		return "", err
	}
	if pkg.ImportPath != "." {
		return pkg.ImportPath, nil
	}
	return "", fmt.Errorf("invalid import path: %q", pkg.ImportPath)
}

func tryGoList(wd string) (string, error) {
	data, err := exec.Command("go", "list", "-f", "{{.ImportPath}}", wd).CombinedOutput()
	out := strings.TrimSpace(string(data))
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return out, nil
}

func isSubdir(root, child string) (string, bool) {
	const sep = string(os.PathSeparator)
	if !strings.HasSuffix(root, sep) {
		root += sep
	}
	if strings.HasPrefix(child, root) {
		return filepath.ToSlash(strings.TrimPrefix(child, root)), true
	}
	return "", false
}

func tryGOPATH(wd string) (string, error) {
	wd = filepath.Clean(wd)
	srcDirs := build.Default.SrcDirs()
	for _, dir := range srcDirs {
		if sub, ok := isSubdir(dir, wd); ok {
			return sub, nil
		}
	}
	return "", fmt.Errorf("directory %q is not contianed in Go source directories: %q", wd, srcDirs)
}

func realMain() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fns := []func(string) (string, error){
		tryGoBuild,
		tryGoList,
		tryGOPATH,
	}
	var first error
	for _, fn := range fns {
		path, err := fn(wd)
		if err == nil {
			return path, nil
		}
		if first == nil {
			first = err
		}
	}
	return "", first
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
