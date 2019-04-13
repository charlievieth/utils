package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// func isSpace(r byte) bool {
// 	return r == ' ' || r == '\t' || r == '\v' || r == '\f' || r == '\r'
// }

func isSpace(r rune) bool {
	return r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r' ||
		r == ' ' || r == 0x85 || r == 0xA0
}

func isComment(line []byte) bool {
	line = bytes.TrimLeftFunc(line, isSpace)
	return len(line) != 0 && line[0] == '#'
}

// func trimLine(line []byte) []byte {
// 	i := 0
// 	for ; i < len(line) && isSpace(line[i]); i++ {
// 	}
// 	return line
// }

func main() {
	{
		fmt.Println(isComment([]byte("# foo")))
		fmt.Println(isComment([]byte("    # foo")))
		fmt.Println(isComment([]byte("    foo")))
		return
	}

	f, err := os.Open(".gitignore")
	if err != nil {
		Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

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
