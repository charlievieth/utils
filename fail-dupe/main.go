// fail-dupe fails a process if it produces duplicate output.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Writer struct {
	w    io.Writer
	cmd  *exec.Cmd
	seen map[string]bool
	buf  []byte
}

func NewWriter(cmd *exec.Cmd, w io.Writer) *Writer {
	return &Writer{
		w:    w,
		cmd:  cmd,
		seen: make(map[string]bool, 1024),
	}
}

func (w *Writer) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		n := bytes.IndexByte(w.buf, '\n')
		if n == -1 {
			break
		}
		line := w.buf[:n+1]
		w.buf = w.buf[n+1:]
		s := string(bytes.TrimSpace(line))
		if len(s) != 0 {
			if w.seen[s] {
				w.Fatalf("duplicate line: %s\n", s)
			} else {
				w.seen[s] = true
			}
		}
		if _, err := w.w.Write(line); err != nil {
			w.Fatalf("error: %s\n", err)
		}
	}
	return len(p), nil
}

func (w *Writer) Fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	if w.cmd.Process != nil {
		if err := w.cmd.Process.Kill(); err != nil {
			fmt.Fprintf(os.Stderr, "killing command (%#v): %s\n", w.cmd, err)
		}
	}
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "Invalid number of args: 0")
		os.Exit(1)
	}
	var cmdArgs []string
	if len(os.Args) > 2 {
		cmdArgs = os.Args[2:]
	}
	cmd := exec.Command(os.Args[1], cmdArgs...)
	stdout := NewWriter(cmd, os.Stdout)
	stderr := NewWriter(cmd, os.Stderr)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "running command (%#v): %s\n", cmd, err)
	}
}
