package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/charlievieth/utils/pathutils"
)

type Result struct {
	Lines []string
	Err   error
}

func Collect(r *pathutils.Reader, delim byte, stop <-chan struct{}) ([]string, error) {
	var err error
	lines := make([]string, 0, 128)
Loop:
	for {
		select {
		case <-stop:
			break Loop
		default:
			// Try to read multiple lines before checking the channel
			// for i := 0; i < 2; i++ {
			b, e := r.ReadBytes(delim)
			if len(b) != 0 {
				lines = append(lines, string(b))
			}
			if e != nil {
				if e != io.EOF {
					err = e
				}
				break Loop
			}
			// }
		}
	}
	sort.Strings(lines)
	return lines, err
}

func PrintLines(w io.Writer, lines []string) error {
	b := bufio.NewWriter(w)
	for _, s := range lines {
		if _, err := b.WriteString(s); err != nil {
			return err
		}
		b.WriteByte('\n')
	}
	return b.Flush()
}

func main() {
	defer os.Stdout.Close()

	stop := make(chan struct{})
	done := make(chan struct{})
	b := bufio.NewReaderSize(os.Stdin, 8192)
	r := pathutils.NewReader(b)

	var lines []string
	var err error
	go func() {
		lines, err = Collect(r, 0, stop)
		close(done)
	}()

	to := time.After(time.Millisecond * 500)
	var timeout bool
	select {
	case <-done:
	case <-to:
		stop <- struct{}{}
		<-done
		timeout = true
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: read: %s\n", err)
	}
	if err := PrintLines(os.Stdout, lines); err != nil {
		fmt.Fprintf(os.Stderr, "Error: print: %s\n", err)
	}
	if !timeout {
		return
	}

	// stream the remaining lines
	w := bufio.NewWriter(os.Stdout)
	for {
		b, e := r.ReadBytes(0)
		if len(b) != 0 {
			w.Write(b)
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: read: %s\n", err)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: print: %s\n", err)
	}

	for _, s := range lines {
		if _, err = w.WriteString(s); err != nil {
			break
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: read: %s\n", err)
	}
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
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
