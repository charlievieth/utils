package main

import (
	"bytes"
	"fmt"
	"os"
)

func main() {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b := bytes.TrimSpace(buf.Bytes())
	buf.Reset()
	buf.Write(b)
	if _, err := buf.WriteTo(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
