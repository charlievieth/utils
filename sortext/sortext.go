package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
)

func main() {
	Run()
}

func Run() {
	lines, err := ReadLines(os.Stdin)
	if err != nil {
		Fatal(err)
	}
	sort.Sort(Bytes(lines))
	sort.Stable(ByExt(lines))
	buf := bufio.NewWriter(os.Stdout)
	for _, b := range lines {
		if _, err := buf.Write(b); err != nil {
			Fatal(err)
		}
		if err := buf.WriteByte('\n'); err != nil {
			Fatal(err)
		}
	}
	if err := buf.Flush(); err != nil {
		Fatal(err)
	}
}

func ReadLines(r io.Reader) ([][]byte, error) {
	lines := make([][]byte, 0, 8)
	buf := bufio.NewReader(r)
	for {
		switch b, err := buf.ReadBytes('\n'); err {
		case nil:
			if include(b) {
				lines = append(lines, b[:len(b)-1])
			}
		case io.EOF:
			if include(b) {
				lines = append(lines, b[:len(b)-1])
			}
			return lines, nil
		default:
			return nil, err
		}
	}
	return lines, nil
}

func include(b []byte) bool {
	return len(b) != 0 && len(bytes.TrimSpace(b)) != 0
}

func PrintStringSlice(s []string) {
	for i := 0; i < len(s); i++ {
		fmt.Println(s[i])
	}
}

type Bytes [][]byte

func (b Bytes) Len() int      { return len(b) }
func (b Bytes) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b Bytes) Less(i, j int) bool {
	return bytes.Compare(b[i], b[j]) == -1
}

type ByExt [][]byte

func (b ByExt) Len() int      { return len(b) }
func (b ByExt) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByExt) Less(i, j int) bool {
	return bytes.Compare(Ext(b[i]), Ext(b[j])) == -1
}

func Ext(path []byte) []byte {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return nil
}

func Fatal(err interface{}) {
	_, file, line, _ := runtime.Caller(1)
	switch e := err.(type) {
	case nil:
		return // Ignore
	case error, string:
		fmt.Fprintf(os.Stderr, "Error (%s:%d): %s\n", file, line, e)
	default:
		fmt.Fprintf(os.Stderr, "Error (%s:%d): %#v\n", file, line, e)
	}
	os.Exit(1)
}
