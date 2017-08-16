package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func knownExt(s string) bool {
	if len(s) != 0 {
		if s[0] != '.' {
			s = filepath.Ext(s)
		}
		return s == ".m4p" || s == ".m4a" || s == ".mp3"
	}
	return false
}

func main() {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("ffprobe", "/tmp/xtmp/i.mp3")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Println("STDOUT:")
	fmt.Println(stdout.String())
	fmt.Println("\nSTDERRR:")
	fmt.Println(stderr.String())
	if err != nil {
		Fatal(err)
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	_, file, line, ok := runtime.Caller(1)
	if ok {
		file = filepath.Base(file)
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		if ok {
			fmt.Fprintf(os.Stderr, "Error (%s:%d): %s", file, line, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s", err)
		}
	default:
		if ok {
			fmt.Fprintf(os.Stderr, "Error (%s:%d): %#v\n", file, line, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
		}
	}
	os.Exit(1)
}
