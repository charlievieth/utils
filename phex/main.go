package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	r := bufio.NewReaderSize(os.Stdin, 16*1024)
	w := bufio.NewWriterSize(os.Stdout, 16*1024)
	out := make([]byte, 80)
	buf := make([]byte, 40)
	var err error
	for {
		n, e := r.Read(buf)
		if n != 0 {
			o := hex.Encode(out, buf[:n])
			if _, e := w.Write(out[:o]); e != nil {
				err = e
				break
			}
			if e := w.WriteByte('\n'); e != nil {
				err = e
				break
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if err != nil {
		Fatal(err) // TODO: replace
	}
	if err := w.Flush(); err != nil {
		Fatal(err)
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
