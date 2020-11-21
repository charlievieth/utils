package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	NoCompression      = 0
	BestSpeed          = 1
	BestCompression    = 9
	DefaultCompression = 6
)

func Enabled() error {
	const input = "H4sIAAAAAAAAA8tIzcnJ5wIAIDA6NgYAAAA="

	path, err := exec.LookPath("pigz")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "--decompress", "--test", "-")
	cmd.Stdin = base64.NewDecoder(base64.StdEncoding, strings.NewReader(input))
	if _, err := cmd.Output(); err != nil {
		e := err.(*exec.ExitError)
		fmt.Println("STDERR:", string(e.Stderr))
		return err // TODO: return an actual error
	}
	return nil
}

func HasPigz() bool {
	const msg = "hello"
	path, err := exec.LookPath("pigz")
	if err != nil {
		return false
	}

	cmd := exec.Command(path, "-c", "-")
	cmd.Stdin = strings.NewReader(msg)

	out, err := cmd.Output()
	if err != nil {
		return false
	}

	gr, err := gzip.NewReader(bytes.NewReader(out))
	if err != nil {
		return false
	}

	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(gr, buf); err != nil {
		return false
	}
	return string(buf) == msg
}

type Pigz struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stderr    *bytes.Buffer
	err       error
	errCh     chan error
	startOnce sync.Once
	closeOnce sync.Once
	level     int
}

func (p *Pigz) close() {
	const timeout = time.Second
	if p.cmd == nil {
		p.err = errors.New("pigz: exec.Command not initialized")
		return
	}
	if p.cmd.Process == nil {
		p.err = errors.New("pigz: internal error: nil process")
		return
	}
	defer p.cmd.Process.Kill()
	if e := p.stdin.Close(); e != nil {
		if p.err == nil && !errors.Is(e, os.ErrClosed) {
			p.err = e
			return // WARN: we shouldn't need this
		}
	}
	to := time.NewTimer(timeout)
	select {
	case p.err = <-p.errCh:
		break
	case <-to.C:
		p.err = errors.New("pigz: timeout waiting for process to exit")
	}
	to.Stop()
}

func (p *Pigz) Close() error {
	p.closeOnce.Do(p.close)
	return p.err
}

func (p *Pigz) wait() {
	err := p.cmd.Wait()
	if err != nil {
		stderr := bytes.TrimSpace(p.stderr.Bytes())
		i := bytes.IndexByte(stderr, '\n')
		if i == -1 {
			i = len(stderr)
		}
		err = fmt.Errorf("pigz: exited with error: %w: %s", err, stderr[:i])
	}
	p.errCh <- err
}

func (p *Pigz) start() {
	if err := p.cmd.Start(); err != nil {
		p.err = err
		return
	}
	go p.wait()
}

func (p *Pigz) Write(b []byte) (int, error) {
	p.startOnce.Do(p.start)
	if p.err != nil {
		return 0, p.err
	}
	n, err := p.stdin.Write(b)
	if n != len(b) {
		err = io.ErrShortWrite
	}
	if err != nil {
		p.Close()
		p.err = err // WARN: do we want to overwrite this ???
	}
	return n, p.err
}

func NewWriter(w io.Writer) (*Pigz, error) {
	return NewWriterLevel(w, DefaultCompression)
}

var ErrNotFound = errors.New("pigz: executable file not found in $PATH")

func (p *Pigz) init(w io.Writer, level int) error {
	// TODO: should this be an error?
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}

	path, err := exec.LookPath("pigz")
	if err != nil {
		return ErrNotFound
	}

	cmd := exec.Command(path, fmt.Sprintf("-%d", level), "-c", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("pigz: creating stdin pipe: %w", err)
	}
	p.stderr.Reset()
	cmd.Stderr = p.stderr // TODO: make value
	cmd.Stdout = w

	*p = Pigz{
		cmd:   cmd,
		stdin: stdin,
		level: level,
		errCh: make(chan error, 1),
	}
	return nil
}

func NewWriterLevel(w io.Writer, level int) (*Pigz, error) {
	if level < NoCompression || level > BestCompression {
		return nil, fmt.Errorf("pigz: invalid compression level: %d", level)
	}
	path, err := exec.LookPath("pigz")
	if err != nil {
		return nil, err
	}
	stderr := new(bytes.Buffer)
	cmd := exec.Command(path, fmt.Sprintf("-%d", level), "-c", "-")
	cmd.Stderr = stderr
	cmd.Stdout = w
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	p := &Pigz{
		cmd:    cmd,
		stdin:  stdin,
		errCh:  make(chan error, 1),
		stderr: stderr,
		level:  level,
	}
	return p, nil
}

// TODO: use this to limit how much we write to STDERR
type limitedWriter struct {
	buf []byte
	off int
}

func newLimitedWriter(size int) *limitedWriter {
	return &limitedWriter{buf: make([]byte, size)}
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.off < len(w.buf) {
		w.off += copy(w.buf[w.off:], p)
	}
	return len(p), nil
}

func (w *limitedWriter) Len() int {
	return w.off
}

func (w *limitedWriter) Reset() {
	w.off = 0
}

func (w *limitedWriter) Bytes() []byte {
	return w.buf[:w.off:w.off]
}
