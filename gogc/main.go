package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {

	// type WalkFunc func(path string, info os.FileInfo, err error) error
	var dirnames []string
	if len(os.Args) == 1 {
		wd, err := os.Getwd()
		if err != nil {
			Fatal(err)
		}
		dirnames = append(dirnames, wd)
	} else {
		for _, s := range os.Args[1:] {
			path, err := filepath.Abs(s)
			if err != nil {
				Fatal(err)
			}
			dirnames = append(dirnames, path)
		}
	}
	for _, root := range dirnames {
		filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
			if fi.Name() != ".git" {
				return nil
			}
			dir := filepath.Dir(path)
			cmd := exec.Command("git", "gc", "--aggressive")
			cmd.Dir = dir
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			fmt.Println("###:", dir)
			if err := cmd.Run(); err != nil {
				Fatal(err)
			}
			return filepath.SkipDir
		})
	}
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
