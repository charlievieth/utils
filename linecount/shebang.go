package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
)

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

func trimSpaceLeft(s []byte) []byte {
	i := 0
	for ; i < len(s); i++ {
		if !isSpace(s[i]) {
			break
		}
	}
	return s[i:]
}

func trimSpaceRight(s []byte) []byte {
	i := len(s)
	for ; i > 0; i-- {
		// include newline
		if c := s[i-1]; !isSpace(c) && c != '\n' {
			break
		}
	}
	return s[:i]
}

func indexSpace(s []byte) int {
	for i := 0; i < len(s); i++ {
		if isSpace(s[i]) {
			return i
		}
	}
	return -1
}

// isProgramName returns if s is likely the name of a program and not
// an option ('-foo') or a shell variable assignment (PATH=/opt/bin:${PATH})
func isProgramName(s []byte) bool {
	return len(s) > 0 && s[0] != '-' && !bytes.Contains(s, []byte("="))
}

func basename(path []byte) string {
	if runtime.GOOS == "windows" {
		// TODO: why is this required for Windows?
		s := filepath.Base(string(path))
		if s == "." || s == string(filepath.Separator) {
			s = ""
		}
		return s
	}
	if len(path) == 0 {
		return ""
	}
	// Strip trailing slashes.
	for len(path) > 0 && os.IsPathSeparator(path[len(path)-1]) {
		path = path[0 : len(path)-1]
	}
	// Find the last element
	i := len(path) - 1
	for i >= 0 && !os.IsPathSeparator(path[i]) {
		i--
	}
	if i >= 0 {
		path = path[i+1:]
	}
	// If empty now, it had only slashes.
	if len(path) == 0 {
		return ""
	}
	return string(path)
}

func parseShebang(line []byte) (string, bool) {
	if len(line) < len("#!/a") || string(line[:2]) != "#!" {
		return "", false
	}

	// Strip off shebang
	s := bytes.TrimSpace(line[len("#!"):])
	if len(s) == 0 || s[0] != '/' {
		return "", false
	}

	// Remove trailing comments, if any
	if i := bytes.IndexByte(s, '#'); i != -1 {
		s = trimSpaceRight(s[:i])
	}

	i := indexSpace(s)

	// Absolute path
	if i == -1 {
		if base := basename(s); base != "" {
			return base, true
		}
		return "", false
	}

	// Special case for `/usr/bin/env CMD`
	if bytes.HasPrefix(s, []byte("/usr/bin/env ")) {
		args := trimSpaceLeft(s[len("/usr/bin/env "):])

		// First argument does not look like an option
		if len(args) > 0 && args[0] != '-' {
			first := args
			if i := indexSpace(first); i != -1 {
				first = first[:i]
			}
			if base := basename(first); base != "" {
				return base, true
			}
			return "", false
		}

		// Return first argument to `env` that is not an option or
		// a shell variable assignment
		for _, a := range bytes.Fields(args) {
			if isProgramName(a) {
				if base := basename(a); base != "" {
					return base, true
				}
				return "", false
			}
		}

		// Failed to parse the arguments to `env`
		return "env", true
	}

	// There are spaces in the interpreter line (`#!/bin/bash --norc`)
	// so just return the name of the first program that will executed
	// this breaks for things like `#!/usr/bin/which python` but that
	// is non-standard and there are too many potential cases to handle.
	s = s[:i]

	if base := basename(s); base != "" {
		return base, true
	}
	return "", false
}

// TODO: stop reading after the initial run of comments
func extractShebang(s []byte) string {
	if i := bytes.IndexByte(s, '\n'); i != -1 {
		s = s[:i]
	}
	prog, _ := parseShebang(s)
	return prog
}
