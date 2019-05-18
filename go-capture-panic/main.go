package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const panicPrefix = "panic: "

var panicBytes = []byte(panicPrefix)

type Reader struct {
	b   *bufio.Reader
	buf []byte

	trace     []byte
	panicking bool
}

func (r *Reader) Panicking() bool { return r.panicking }

func (r *Reader) Trace() []byte {
	if len(r.trace) == 0 {
		return nil
	}
	// TODO: reserve space to save an alloc
	return append(bytes.ReplaceAll(r.trace, []byte{'\n'}, []byte("\\n")), '\n')
}

func (r *Reader) ReadBytes(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	r.buf = r.buf[:0]
	for {
		var e error
		frag, e = r.b.ReadSlice(delim)
		if e == nil { // final fragment
			break
		}
		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}
		r.panicking = r.panicking || len(frag) >= len(panicPrefix) &&
			bytes.HasPrefix(frag, panicBytes)
		// TODO(cev): cap panic space or use a sufficiently large
		// circular buffer. We currently assume that a panic will
		// be shortly followed by EOF.
		if r.panicking {
			r.trace = append(r.trace, frag...)
		} else {
			r.buf = append(r.buf, frag...)
		}
	}
	if r.panicking {
		r.trace = append(r.trace, frag...)
	} else {
		r.buf = append(r.buf, frag...)
	}
	return r.buf, err
}

// CEV: assume stderr for now
func (r *Reader) Copy() (err error) {
	for {
		b, er := r.ReadBytes('\n')
		if len(b) != 0 {
			if _, ew := os.Stderr.Write(b); ew != nil {
				if er == nil {
					er = ew
				}
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	// Prepare to dump panic trace, if any.
	if len(r.trace) != 0 {
		if _, ew := os.Stderr.Write(r.Trace()); ew != nil {
			if err == io.EOF {
				err = ew
			}
		}
	}
	return err
}

func realMain(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout

	return nil
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "Error: missing args") // TODO: print usage
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	if err := realMain(cmd, args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
