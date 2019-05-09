package main

import (
	"encoding/ascii85"
	"flag"
	"io"
	"os"
)

var (
	InputFile  string
	OutputFile string
	LineBreak  int
)

func init() {
	flag.StringVar(&InputFile, "i", "", "Input file")
	flag.StringVar(&OutputFile, "o", "", "Output file")
	flag.IntVar(&LineBreak, "b", 0, "Line break")
}

type BreakWriter struct {
	w       io.Writer
	n       int
	buf     []byte // leftover
	scratch []byte // length == n + 1
	// buf     bytes.Buffer // leftover
}

func (w *BreakWriter) Write(p []byte) (int, error) {
	if w.scratch == nil {
		w.scratch = make([]byte, w.n+1)
		w.scratch[len(w.scratch)-1] = '\n'
	}
	b := append(w.buf, p...)
	var err error
	for len(b) >= w.n {
		copy(w.scratch, b[0:w.n])
		b = b[w.n:]
		if _, err = w.w.Write(w.scratch); err != nil {
			break
		}
	}
	if len(b) != 0 {
		w.buf = append(w.buf[:0], b...)
	}
	return len(p), err
}

/*
func (w *BreakWriter) Write(p []byte) (int, error) {
	if w.scratch == nil {
		w.scratch = make([]byte, w.n+1)
		w.scratch[len(w.scratch)-1] = '\n'
	}
	scratch := w.scratch[:len(w.scratch)-1]
	if len(scratch) != w.n {
		panic("WAT") // sanity check - remove
	}
	total := 0
	w.buf.Write(p)
	for w.buf.Len() >= w.n {
		n, er := w.buf.Read(scratch)
		total += n
		if n == len(scratch) {
			_, ew := w.w.Write(w.scratch) // w.scratch has the newline
			if ew != nil {
				return total, ew
			}
		}
		if er != nil {

		}
	}

	// var b []byte
	// if len(w.buf) != 0 {
	// 	b = append(b, p...) // lazy and wasteful
	// }
	return len(p), nil
}
*/

func (w *BreakWriter) Close() (err error) {
	if len(w.buf) != 0 {
		// _, err = w.w.Write(w.buf)
		// w.buf = w.buf[:0]
	}
	return err
}

func realMain() error {
	flag.Parse()

	var in io.Reader
	if InputFile != "" {
		f, err := os.Open(InputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	} else {
		in = os.Stdin
	}

	var out io.Writer
	if OutputFile != "" {
		f, err := os.OpenFile(OutputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	w := ascii85.NewEncoder(out)
	if _, err := io.Copy(w, in); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func main() {

}
