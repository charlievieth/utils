package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func formatJSON(r io.Reader) ([]byte, error) {
	var v interface{}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "    ")
}

func main() {
	var v interface{}
	if err := json.NewDecoder(os.Stdin).Decode(v); err != nil {
		Fatal(err)
	}
	b, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		Fatal(err)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		Fatal(err)
	}
	// json.MarshalIndent(v, prefix, indent)
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
