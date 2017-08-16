package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough args")
		os.Exit(1)
	}
	size, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	size *= 1024 * 1024
	line := append(bytes.Repeat([]byte{'A'}, 127), '\n')
	for i := 0; i < size; i += len(line) {
		_, err := os.Stdout.Write(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}
