package main

import (
	"os"
	"strings"
)

func combine(args ...string) string {
	var paths []string
	for _, s := range args {
		paths = append(paths, strings.Split(s, string(os.PathListSeparator))...)
	}
	seen := make(map[string]bool, len(paths))
	a := paths[:0]
	for _, s := range paths {
		if !seen[s] {
			seen[s] = true
			a = append(a, s)
		}
	}
	return strings.Join(a, string(os.PathListSeparator))
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		return // ERROR
	}
	if len(args) == 1 {
		os.Stdout.WriteString(args[0])
		return
	}
	os.Stdout.WriteString(combine(args...))
}
