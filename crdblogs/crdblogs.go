package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/charlievieth/reonce"
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
	if n := len(frag); n != 0 && frag[n-1] == delim {
		frag = frag[:n-1]
	}
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

func IndentJSON(src []byte) []byte {
	var dst bytes.Buffer
	if err := json.Indent(&dst, src, "", "    "); err != nil {
		return src
	}
	return dst.Bytes()
}

type Line struct {
	Base  []byte
	JSON  []byte
	Extra []byte // color
}

func (l *Line) Format(dst *bytes.Buffer) error {
	dst.Write(l.Base)
	if len(l.JSON) != 0 {

		dst.Write(IndentJSON(l.JSON))
	}
	if len(l.Extra) != 0 {
		dst.Write(l.Extra)
	}
	return nil
}

func (l *Line) WriteTo(w io.Writer) (int64, error) {
	var nn int64
	if len(l.Base) != 0 {
		n, err := w.Write(l.Base)
		if err != nil {
			return nn, err
		}
		nn += int64(n)
	}
	if len(l.JSON) != 0 {
		n, err := w.Write(IndentJSON(l.JSON))
		if err != nil {
			return nn, err
		}
		nn += int64(n)
	}
	if len(l.Extra) != 0 {
		n, err := w.Write(l.Extra)
		if err != nil {
			return nn, err
		}
		nn += int64(n)
	}
	return 0, nil
}

func ParseLine(b []byte) *Line {
	i := bytes.IndexByte(b, '{')
	if i == -1 {
		return &Line{Base: b}
	}
	j := bytes.LastIndexByte(b, '}')
	if j == -1 {
		return &Line{Base: b}
	}
	return &Line{Base: b[:i], JSON: b[i : j+1], Extra: b[j+1:]}
}

var AnsiRe = reonce.New("\x1b" + `\[\d+(?:;\d+)*m`)

func VisibleLength(s string) int {
	if strings.Contains(s, "\x1b") {
		s = AnsiRe.ReplaceAllString(s, "")
	}
	return utf8.RuneCountInString(s)
}

const ExampleLine = "\x1b[38;5;33mI211021 \x1b[38;5;246m16:05:38.902051 16753 tenantusage/controller_integration_test.go:115\x1b[0m  AdminClient.LimitTenantUsage \x1b[38;5;4m{\"service\":\"tenantusage\",\"dd.trace_id\":\"0\"}\x1b[0m\r\n"

func trimTrailingWhitespace(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		c := b[i]
		switch c {
		case ' ', '\t', '\n', '\r', '\v':
		default:
			return b[:i+1]
		}
	}
	return b[:0]
}

func ProcessStdin() error {
	const prefix = "        " + "        " // 16
	const indent = "    "
	var dst bytes.Buffer
	var buf bytes.Buffer
	r := NewReader(bufio.NewReader(os.Stdin))
	var err error
	for {
		b, e := r.ReadBytes('\n')
		b = trimTrailingWhitespace(b)
		if len(b) != 0 {
			buf.Reset()
			ll := ParseLine(b)
			buf.Write(ll.Base)
			if len(ll.JSON) != 0 {
				dst.Reset()
				if err := json.Indent(&dst, ll.JSON, prefix, indent); err != nil {
					buf.Write(ll.JSON)
				} else {
					buf.WriteByte('\n')
					buf.WriteString(prefix)
					buf.Write(dst.Bytes())
					// buf.WriteByte('\n')
				}
			}
			if len(ll.Extra) != 0 {
				buf.Write(ll.Extra)
			}
			buf.WriteByte('\n')
			if _, err := buf.WriteTo(os.Stdout); err != nil {
				if e == nil && e != io.EOF {
					e = err
				}
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
		return err
	}
	return nil
}

func main() {
	if err := ProcessStdin(); err != nil {
		Fatal(err)
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
