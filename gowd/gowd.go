package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func tryGoBuild(_ string) (string, error) {
	pkg, err := build.ImportDir(".", 0)
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

func fileExists(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.Mode().IsRegular()
}

func findModule(root string) (string, error) {
	dir := root
	for i := 0; i < 64; i++ {
		mod := filepath.Join(dir, "go.mod")
		if fileExists(mod) {
			return mod, nil
		}
		next := filepath.Base(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return "", fmt.Errorf("go.mod not found in tree: %s", root)
}

// TODO: this does not appear to always work
func tryModule(wd string) (string, error) {
	mod, err := findModule(wd)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(mod)
	if err != nil {
		return "", err
	}
	pkgPath := modfile.ModulePath(data)
	if pkgPath == "" {
		return "", fmt.Errorf("failed to parse module path from: %s", mod)
	}
	rel, err := filepath.Rel(filepath.Dir(mod), wd)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return pkgPath, nil
	}
	return filepath.ToSlash(pkgPath + "/" + rel), nil
}

func gowdForDir(dir string) (string, error) {
	fns := []func(string) (string, error){
		tryModule,
		tryGoBuild,
		tryGoList,
		tryGOPATH,
	}
	var first error
	for _, fn := range fns {
		path, err := fn(dir)
		if err == nil {
			return path, nil
		}
		if first == nil {
			first = err
		}
	}
	return "", first
}

func realMain() error {
	var dirs []string
	for _, arg := range flag.Args() {
		fi, err := os.Stat(arg)
		if err != nil {
			continue
		}
		if fi.Mode().IsRegular() {
			arg = filepath.Dir(arg)
		}
		abs, err := filepath.Abs(arg)
		if err != nil {
			continue
		}
		dirs = append(dirs, abs)
	}
	if len(dirs) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dirs = append(dirs, wd)
	}
	var first error
	for _, dir := range dirs {
		gp, err := gowdForDir(dir)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		// Immediately return print errors since it means we cannot continue.
		if _, err := fmt.Println(gp); err != nil {
			return err
		}
	}
	return first
}

func main() {
	flag.Usage = func() {
		const msg = "gowd: [PATH...]\n" +
			"print the import path of the Go package corresponding to PATH(s) or the current directory\n"
		fmt.Fprint(flag.CommandLine.Output(), msg)
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
