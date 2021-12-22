package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Line struct {
	Raw     string
	NoColor string
}

type lineByNoColor struct {
	Lines []Line
}

func (ll *lineByNoColor) Len() int {
	return len(ll.Lines)
}

func (ll *lineByNoColor) Less(i, j int) bool {
	return ll.Lines[i].NoColor < ll.Lines[j].NoColor
}

func (ll *lineByNoColor) Swap(i, j int) {
	ll.Lines[i], ll.Lines[j] = ll.Lines[j], ll.Lines[i]
}

var replaceRe = regexp.MustCompile("\x1b" + `\[[0-9][^m]*m`)

func ReplaceANSII(s []byte) []byte {
	if bytes.IndexByte(s, '\x1b') != -1 {
		return replaceRe.ReplaceAll(s, nil)
	}
	b := make([]byte, len(s))
	copy(b, s)
	return b
}

func ReplaceANSIIString(s string) string {
	if strings.IndexByte(s, '\x1b') != -1 {
		return replaceRe.ReplaceAllString(s, "")
	}
	return s
}

type Result struct {
	Lines []string
	Err   error
}

type Reader struct {
	b   *bufio.Reader
	buf []byte
}

func NewReader(b *bufio.Reader) *Reader {
	return &Reader{
		b:   b,
		buf: make([]byte, 128),
	}
}

func (r *Reader) ReadBytes(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	r.buf = r.buf[:0]
	for {
		var e error
		frag, e = r.b.ReadSlice(delim)
		if e == nil { // got final fragment
			break
		}
		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}
		r.buf = append(r.buf, frag...)
	}
	// Include delim in the returned line
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

func trimSpaceLeft(s []byte) []byte {
	if n := len(s); n != 0 && s[n-1] == ' ' {
		return s[:n-1]
	}
	return s
}

func Collect(r *Reader, delim byte, ignoreCase bool, stop <-chan struct{}) ([]Line, bool, error) {
	var timeout bool
	var err error
	lines := make([]Line, 0, 128)

Loop:
	for i := 1; ; i++ {
		b, e := r.ReadBytes(delim)
		if len(b) != 0 {
			raw := string(b)
			noColor := ReplaceANSIIString(raw)
			if ignoreCase {
				noColor = strings.ToLower(noColor)
			}
			lines = append(lines, Line{
				Raw:     raw,
				NoColor: noColor,
			})
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break Loop
		}
		// check stop every 128 lines
		if i%128 == 0 && stop != nil {
			select {
			case <-stop:
				timeout = true
				break Loop
			default:
			}
		}
	}

	return lines, timeout, err
}

// TODO: support line buffering
func PrintLines(w io.Writer, lines []Line) error {
	b, ok := w.(*bufio.Writer)
	if !ok {
		b = bufio.NewWriterSize(w, 8192)
	}
	for _, line := range lines {
		if _, err := b.WriteString(line.Raw); err != nil {
			return err
		}
	}
	return b.Flush()
}

func fixupLines(lines []Line, ignoreCase bool) []Line {
	numWorkers := len(lines) / 4096
	if numWorkers <= 1 {
		for i := range lines {
			nocolor := ReplaceANSIIString(lines[i].Raw)
			if ignoreCase {
				nocolor = strings.ToLower(nocolor)
			}
			lines[i].NoColor = nocolor
		}
	}
	if numWorkers > runtime.NumCPU() {
		numWorkers = runtime.NumCPU()
	}

	var wg sync.WaitGroup
	gate := make(chan struct{}, numWorkers)
	for i := 0; i < len(lines); i += 4096 {
		wg.Add(1)
		go func(lines []Line) {
			gate <- struct{}{}
			defer func() {
				wg.Done()
				<-gate
			}()
			for i := range lines {
				nocolor := ReplaceANSIIString(lines[i].Raw)
				if ignoreCase {
					nocolor = strings.ToLower(nocolor)
				}
				lines[i].NoColor = nocolor
			}
		}(lines[i : i+4096])
	}
	wg.Wait()

	return lines
}

func main() {
	nullDelim := pflag.BoolP("null", "0", false, "input is NUL terminated")
	pflag.Parse()
	ignoreCase := pflag.BoolP("ignore-case", "f", false, "ignore case when sorting")
	defer os.Stdout.Close()

	stop := make(chan struct{})
	done := make(chan struct{})
	b := bufio.NewReaderSize(os.Stdin, 8192)
	r := NewReader(b)

	var lines []Line
	var err error
	delim := byte('\n')
	if *nullDelim {
		delim = 0
	}
	var timeout bool
	go func(delim byte) {
		lines, timeout, err = Collect(r, delim, *ignoreCase, stop)
		close(done)
	}(delim)

	to := time.After(1)
	// to := time.After(time.Millisecond * 500)
	select {
	case <-done:
	case <-to:
		stop <- struct{}{}
		<-done
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: read: %s\n", err)
	}

	if *ignoreCase {
		for i := range lines {
			lines[i].NoColor = strings.ToLower(lines[i].NoColor)
		}
	}
	sort.Sort(&lineByNoColor{Lines: lines})
	if err := PrintLines(os.Stdout, lines); err != nil {
		fmt.Fprintf(os.Stderr, "Error: print: %s\n", err)
	}

	if !timeout {
		return
	}

	// stream the remaining lines
	w := bufio.NewWriter(os.Stdout)
	for {
		b, e := r.ReadBytes(delim)
		if len(b) != 0 {
			// WARN: missing new line !!!
			if _, ew := w.Write(b); ew != nil && e == nil {
				e = ew
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
		fmt.Fprintf(os.Stderr, "Error: read: %s\n", err)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: print: %s\n", err)
	}

	for _, line := range lines {
		if _, err = w.WriteString(line.Raw); err != nil {
			break
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: read: %s\n", err)
	}
}

func rootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "fdsort [OPTIONS]...",
		Short: "fdsort", // TODO: fix me
	}
	flags := root.Flags()
	// TODO: use "auto", "always", "never"
	stripSlash := flags.BoolP("strip-slash", "s", false, "strip leading './'")
	noStripSlash := flags.Bool("no-strip-slash", false, "strip leading './'")

	// -0, --null
	nullIn := flags.BoolP("null", "0", false, "input is NUL terminated")
	nullOut := flags.Bool("null-data", false, "use NUL as the line terminator instead of \\n")

	// TODO: note that a negative timeout disables this
	timeout := flags.DurationP("timeout", "t", time.Second/2,
		"timeout to read and sort STDIN, after this STDIN will be "+
			"written to STDOUT unsorted")

	_ = timeout
	_ = nullIn
	_ = nullOut

	root.PreRunE = func(cmd *cobra.Command, args []string) error {
		if *stripSlash && *noStripSlash {
			return errors.New("both `--strip-slash` and `--no-strip-slash` " +
				"flags cannot be defined")
		}
		return nil
	}
	root.Run = func(cmd *cobra.Command, args []string) {
		panic("implement!")
	}

	return root
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
