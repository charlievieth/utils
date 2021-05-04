package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func safeMul(a, b uint) uint {
	c := a * b
	if a > 1 && b > 1 && c/b != a {
		return 0
	}
	return c
}

func parseSize(sizeStr string) (uint, error) {
	s := strings.TrimSpace(sizeStr)

	var suffix string
	var size string
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if '0' <= c && c <= '9' {
			suffix = s[i+1:]
			size = s[:i+1]
			break
		}
	}

	if strings.HasSuffix(suffix, "b") || strings.HasSuffix(suffix, "B") {
		suffix = suffix[:len(suffix)-1]
	}

	multiplier := uint64(1)
	switch suffix {
	case "k", "K":
		multiplier = 1 << 10
	case "m", "M":
		multiplier = 1 << 20
	case "g", "G":
		multiplier = 1 << 30
	case "":
		// Ok
	default:
		return 0, errors.New("invalid size suffix: " + suffix)
	}

	u, err := strconv.ParseUint(size, 10, 64)
	if err != nil {
		return 0, err
	}
	u *= multiplier

	return uint(u), nil
}

type Value interface {
	String() string
	Set(string) error
	Type() string
}

type ByteSizeValue struct {
	s string
	n int64
}

func (v *ByteSizeValue) String() string {
	return v.s
}

func (v *ByteSizeValue) Type() string {
	return "ByteSizeValue"
}

func (v *ByteSizeValue) Set(val string) error {
	u, err := parseSize(val)
	if err != nil {
		return err
	}
	v.n = int64(u)
	return nil
}

func main() {
	// pflag.Bool("name", false, "usage")
	// pflag.Value

	glob := flag.String("glob", "*", "Pattern to match")
	_ = glob

	flag.Parse()

	var dirs []string
	if flag.NArg() > 0 {
		for _, s := range flag.Args() {
			dirs = append(dirs, s)
		}
	} else {
		dirs = append(dirs, ".")
	}

	for _, dir := range dirs {
		names, err := filepath.Glob(dir + "/" + *glob)
		if err != nil {
			Fatal(err)
		}
		if len(names) != 0 {
			fmt.Printf("%s:\n", dir)
			for _, s := range names {
				fmt.Printf("    %s\n", filepath.Base(s))
			}
		}
	}
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
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
