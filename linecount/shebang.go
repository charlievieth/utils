package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
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

func indexSpace(s []byte) int {
	for i := 0; i < len(s); i++ {
		if isSpace(s[i]) {
			return i
		}
	}
	return -1
}

func lastIndexSpace(s []byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if isSpace(s[i]) {
			return i
		}
	}
	return -1
}

func maybeContainsOptions(s []byte) bool {
	for i := 0; i < len(s)-1; i++ {
		if isSpace(s[i]) && s[i+1] == '-' {
			return true
		}
	}
	return false
}

var ErrInvalidShebang = errors.New("invalid shebang")

// var optionRe = regexp.MustCompile(`((?:\s+)(-{1,2}[[:alnum:]][-_[:alnum:]]*|--|-$))`)
var optionRe = regexp.MustCompile(`((?:\s+)(-{1,2}[[:alnum:]][^\s]*|--|-$))`)

var shebang = []byte("#!")

func filepathBase(path []byte) string {
	if runtime.GOOS == "windows" {
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

func ParseShebangExe(line []byte) (string, error) {
	s := bytes.TrimSpace(line)
	if !bytes.HasPrefix(s, shebang) {
		return "", ErrInvalidShebang
	}

	// Strip off shebang
	s = trimSpaceLeft(s[len("#!"):])
	if len(s) == 0 || s[0] != '/' {
		return "", ErrInvalidShebang
	}
	if i := bytes.Index(s, []byte("<%=")); i != -1 {
		for i = i - 1; i >= 0 && isSpace(s[i]); i-- {
		}
		s = s[:i+1]
	}

	// Remove trailing comments, if any
	if i := bytes.IndexByte(s, '#'); i != -1 {
		for i = i - 1; i >= 0 && isSpace(s[i]); i-- {
		}
		s = s[:i+1]
	}

	// Remove any flags
	if maybeContainsOptions(s) {
		s = optionRe.ReplaceAll(s, nil)
	}

	i := indexSpace(s)

	// Absolute path
	if i == -1 {
		if base := filepathBase(s); base != "" {
			return base, nil
		}
		return "", ErrInvalidShebang
	}

	s = s[i+1:]
	if i := lastIndexSpace(s); i != -1 {
		return string(s[i+1:]), nil
	}
	if base := filepathBase(s); base != "" {
		return base, nil
	}
	return "", ErrInvalidShebang
}

func ExtractShebang(s []byte) string {
	for {
		m := bytes.Index(s, shebang)
		if m < 0 {
			break
		}
		// extract full line
		start := bytes.LastIndexByte(s[:m], '\n') + 1
		end := bytes.IndexByte(s[m:], '\n')
		if end != -1 {
			end += m
		} else {
			end = len(s)
		}
		// remove leading space
		for ; start < len(s); start++ {
			c := s[start]
			if c != ' ' && c != '\t' {
				break
			}
		}
		if start > end {
			return "" // this should never happen
		}
		line := s[start:end]
		if len(line) > 2 && line[0] == '#' && line[1] == '!' {
			if exe, _ := ParseShebangExe(line); exe != "" {
				if exe == "Python" {
					exe = "python"
				}
				return exe
			}
		}
		s = s[end:]
	}
	return ""
}
