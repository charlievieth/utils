package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

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
	// includes delim
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

type Match struct {
	Filename string
	Line     int
	Column   int
	Raw      string // raw match including newline
}

// Sort by filename => line => column
type byNameLineCol []*Match

func (m byNameLineCol) Len() int      { return len(m) }
func (m byNameLineCol) Swap(i, j int) { m[i], m[j] = m[j], m[i] }

func (m byNameLineCol) Less(i, j int) bool {
	m1 := m[i]
	m2 := m[j]
	// TODO: support invalid matches by comparing Raw
	if m1.Filename < m2.Filename {
		return true
	}
	if m1.Filename > m2.Filename {
		return false
	}
	if m1.Line < m2.Line {
		return true
	}
	if m1.Line > m2.Line {
		return false
	}
	return m1.Column < m2.Column
}

func hasANSIEscapePrefix(s string) bool {
	return strings.HasPrefix(s, "\x1b[")
}

func stripANSI(s string) string {
	if !hasANSIEscapePrefix(s) {
		return s
	}
	var w strings.Builder
	w.Grow(len(s) * 2 / 3) // reserve 2/3 of the length of the string
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\x1b' {
			for ; i < len(s) && s[i] != 'm'; i++ {
			}
		} else {
			w.WriteByte(c)
		}
	}
	if w.Len() == 0 {
		return ""
	}
	return w.String()
}

var (
	errInvalidLineCol = errors.New("invalid line col")
	errInvalidMatch   = errors.New("invalid rg match result")
)

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func parseLineCol(s string) (line, col int, err error) {
	if !strings.HasPrefix(s, ":") {
		return 0, 0, errInvalidLineCol
	}
	s = s[1:]

	// line number
	var i int
	for i = 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			break
		}
	}
	if i == 0 || s[i] != ':' {
		return 0, 0, errInvalidLineCol
	}
	line, err = strconv.Atoi(s[:i])
	if err != nil {
		return 0, 0, err
	}
	s = s[i+1:] // trim line number + ":"

	// column number
	for i = 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			break
		}
	}
	if i == 0 || s[i] != ':' {
		return 0, 0, errInvalidLineCol
	}
	col, err = strconv.Atoi(s[:i])
	if err != nil {
		return 0, 0, err
	}

	return line, col, nil
}

func parseMatch(raw string) (*Match, error) {
	s := raw
	if hasANSIEscapePrefix(s) {
		s = stripANSI(s)
	}
	p := s
	o := 0
	for len(s) > 0 {
		i := strings.Index(s, ":")
		if i == -1 {
			return nil, errInvalidMatch
		}
		o += i
		line, col, err := parseLineCol(s[i:])
		if err == nil {
			m := &Match{
				Filename: p[:o],
				Line:     line,
				Column:   col,
				Raw:      raw,
			}
			return m, nil
		}
		s = s[i+1:]
		o++
	}
	return nil, errInvalidMatch
}

func isSpace(b []byte) bool {
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case '\t', '\n', '\v', '\f', '\r', ' ':
		default:
			return false
		}
	}
	return true
}

func isRgError(s string) bool {
	return strings.Contains(s, "WARNING:") || strings.Contains(s, "ERROR:")
}

func newInvalidInputError(errs, lines int) error {
	return fmt.Errorf("failed to parse %d of %d lines (%%%.2f), "+
		"did you forget to run `rg` with the `--vimgrep` option?",
		errs, lines, (float64(errs)/float64(lines))*100)
}

func overErrorThreshold(errs, lines int) bool {
	return errs >= 4 && errs == lines ||
		errs >= 16 && errs >= (lines*3)/2
}

func main() {
	root := cobra.Command{
		Use:     "rgsort",
		Short:   "rgsort: sort the output of `rg --vimgrep` by filename and line number.",
		Example: "$ rg --vimgrep '^static' | rgosrt",
		Args:    cobra.NoArgs,
	}
	caseSensitive := root.Flags().BoolP("case-sensitive", "s", false,
		"Sort result file names case sensitively.")
	ignoreErrorsP := root.Flags().Bool("ignore-errors", false,
		"Ignore parse errors and don't abort early if there are too many.")

	root.RunE = func(cmd *cobra.Command, args []string) error {
		var (
			err       error
			errCount  int
			lineCount int
			matches   []*Match
		)
		ignoreErrors := *ignoreErrorsP
		r := NewReader(bufio.NewReader(os.Stdin))
		for err == nil {
			var b []byte
			b, err = r.ReadBytes('\n')
			if len(b) != 0 && !isSpace(b) {
				lineCount++
				// TODO: include the Raw portion of invalid matches
				raw := string(b)
				m, merr := parseMatch(raw)
				if merr == nil {
					matches = append(matches, m)
					continue
				}
				// Handle error
				fmt.Fprintf(os.Stderr, "Warning: %v: %q\n", merr, raw)
				if !isRgError(raw) {
					errCount++
					if overErrorThreshold(errCount, lineCount) && !ignoreErrors {
						err = newInvalidInputError(errCount, lineCount)
					}
				}
			}
		}
		if err == nil && errCount == lineCount || overErrorThreshold(errCount, lineCount) {
			err = newInvalidInputError(errCount, lineCount)
		}
		if err != io.EOF {
			return fmt.Errorf("reading input: %w\n", err)
		}

		if !*caseSensitive {
			for _, m := range matches {
				m.Filename = strings.ToLower(m.Filename)
			}
		}
		sort.Sort(byNameLineCol(matches))

		w := bufio.NewWriterSize(os.Stdout, 32*1024)
		for _, m := range matches {
			if _, err := w.WriteString(m.Raw); err != nil {
				return fmt.Errorf("writing output: %w\n", err)
			}
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("writing output: %w\n", err)
		}
		return nil
	}

	if root.Execute() != nil {
		os.Exit(1)
	}
}
