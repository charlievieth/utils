package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// TODO:
//  * Enforce max line length
//  * Use a file for intermediate panic storage, this allows
//    us to handle very large panics without allocating
//  * Make all this configurable (including file location)

const (
	panicPrefix = "panic: "

	envPrefix = "GCP_"

	// use '\r\n' as the default newline replacement since it
	// is more distinctive than just '\n'
	defaultNewLineReplacement = "\\r\\n"

	defaultMaxLineLength = 16 * 1014

	// Use a buffer large enough to capture large stack traces in one read.
	// This should help in OOM situations by allowing the crashing program
	// to dump its stack and exit before we format the message.
	defaultReaderSize = 16 * 1024 * 1024
)

var panicPrefixes = [...][]byte{
	[]byte("panic: "),
	[]byte("fatal error: "),
}

var panicBytes = []byte(panicPrefix)
var ErrTooLong = errors.New("go-capture-panic: token too long")

type Scanner struct {
	rd        *Reader
	panicking bool
	trace     []byte
	opts      Options
}

type Options struct {
	Logger             *zap.Logger // TOOD (CEV): make configurable
	MaxLineLength      int
	ReaderSize         int
	NewLineReplacement []byte
	Namespace          string
	SignalChild        bool // send all signals to our child process
}

func envInt(key string) int {
	i, _ := strconv.Atoi(os.Getenv(key))
	return i
}

func envBool(key string) bool {
	b, _ := strconv.ParseBool(os.Getenv(key))
	return b
}

func envBytes(key string) []byte {
	if s := os.Getenv(key); s != "" {
		return []byte(s)
	}
	return nil
}

func EnvOptions() *Options {
	return &Options{
		MaxLineLength:      envInt(envPrefix + "MAX_LINE_LENGTH"),
		ReaderSize:         envInt(envPrefix + "READER_SIZE"),
		NewLineReplacement: envBytes(envPrefix + "NEW_LINE_REPLACEMENT"),
		Namespace:          os.Getenv(envPrefix + "NAMESPACE"),
		SignalChild:        envBool(envPrefix + "SIGNAL_CHILD"),
	}
}

func (o Options) Init(r io.Reader, cmd string, args ...string) *Scanner {
	if o.MaxLineLength == 0 {
		o.MaxLineLength = defaultMaxLineLength
	}
	if o.ReaderSize == 0 {
		o.ReaderSize = defaultReaderSize
	}
	if len(o.NewLineReplacement) == 0 {
		o.NewLineReplacement = []byte(defaultNewLineReplacement)
	}
	if o.Namespace == "" {
		o.Namespace = filepath.Base(cmd)
	}
	// TOOD (CEV): make configurable
	if o.Logger == nil {
		// z := zap.NewProduction().Named(s)

	}
	return &Scanner{
		rd:    NewReaderSize(r, o.ReaderSize, o.MaxLineLength),
		trace: allocBytes(0, 128*1024),
		opts:  o,
	}
}

// TODO: reserve space in case we are in an OOM situation
type Reader struct {
	b         *bufio.Reader
	buf       []byte // line buffer
	trace     []byte // trace buffer
	maxLength int    // max line length (loosely enforced)
	panicking bool
}

func NewReaderSize(r io.Reader, size, maxLength int) *Reader {
	return &Reader{
		b:         allocBufioReader(r, size),
		buf:       make([]byte, 0, 256),
		trace:     make([]byte, 0, 32*1024),
		maxLength: maxLength,
	}
}

func NewReader(r io.Reader) *Reader {
	return NewReaderSize(r, defaultReaderSize, defaultMaxLineLength)
}

func (r *Reader) Panicking() bool { return r.panicking }

// TODO: pass a scratch buffer so that we don't have to allocate
func ReplaceNewLines(s, new, buf []byte) []byte {
	n := bytes.Count(s, []byte{'\n'})
	if n == 0 {
		return append([]byte(nil), s...)
	}

	size := len(s) + n*(len(new)-1) // reserve space for newline
	var t []byte
	if cap(buf) < size {
		t = make([]byte, size, size+1)
	} else {
		t = buf[0:size]
	}
	w := 0
	start := 0
	for i := 0; i < n; i++ {
		j := start
		j += bytes.IndexByte(s[start:], '\n')
		w += copy(t[w:], s[start:j])
		w += copy(t[w:], new)
		start = j + 1
	}
	w += copy(t[w:], s[start:])
	return append(t[0:w], '\n')
}

func (r *Reader) Trace() []byte {
	if len(r.trace) == 0 {
		return nil
	}
	trace := r.trace
	if bytes.Contains(trace, []byte("\r\n")) {
		trace = bytes.ReplaceAll(trace, []byte("\r\n"), []byte{'\n'})
	}
	return ReplaceNewLines(trace, []byte(defaultNewLineReplacement), nil)
}

type LogEntry struct {
	Level     string            `json:"level"`
	Timestamp float64           `json:"ts"`
	Logger    string            `json:"logger"`
	Message   string            `json:"msg"`
	Pid       int               `json:"pid"` // ppid
	JSON      map[string]string `json:"json,omitempty"`
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
		// We still append frag to r.buf even though the result exceeds
		// maxLength otherwise we would lose it.
		if len(r.buf)+len(frag) > r.maxLength {
			err = ErrTooLong
			break
		}
		r.buf = append(r.buf, frag...)
	}
	r.buf = append(r.buf, frag...)
	return r.buf, err
}

// TODO:
// * cap line length
// * consume all of the panic
func (r *Reader) ReadBytesPanic(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	r.buf = r.buf[:0]
	for {
		var e error
		frag, e = r.b.ReadSlice(delim)

		// the panic check must occur before breaking the loop
		if !r.panicking && len(frag) >= len(panicPrefix) {
			// fatal error:
			r.panicking = bytes.HasPrefix(frag, panicBytes)
		}

		if e == nil { // final fragment
			break
		}
		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}

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

// TODO (CEV): rename
func (s *Scanner) Consume(w io.Writer) (err error) {
	for {
		b, e := s.rd.ReadBytes('\n')
		if len(b) != 0 {
			s.panicking = s.panicking || len(b) >= len(panicPrefix) &&
				bytes.HasPrefix(b, panicBytes)
			if !s.panicking {
				if _, ew := w.Write(b); ew != nil && (e == nil || e == io.EOF) {
					e = ew
				}
			} else {
				s.trace = append(s.trace, b...)
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	if len(s.trace) != 0 {

	}
	return err
}

// CEV: assume stderr for now
func (r *Reader) Copy(w io.Writer) (err error) {
	for {
		b, er := r.ReadBytesPanic('\n')
		if len(b) != 0 {
			if _, ew := w.Write(b); ew != nil {
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
		if _, ew := w.Write(r.Trace()); ew != nil {
			if err == io.EOF {
				err = ew
			}
		}
	}
	return err
}

func cleanEnv() []string {
	env := os.Environ()
	a := env[:0]
	for _, s := range env {
		if !strings.HasPrefix(s, envPrefix) {
			a = append(a, s)
		}
	}
	return a
}

var signals = [...]os.Signal{
	syscall.SIGABRT,
	syscall.SIGALRM,
	// syscall.SIGCHLD, // ignore
	syscall.SIGCONT,
	syscall.SIGHUP,
	syscall.SIGINFO,
	syscall.SIGINT,
	// syscall.SIGKILL, // can't catch
	syscall.SIGQUIT,
	syscall.SIGSTOP,
	syscall.SIGTERM,
	// syscall.SIGTSTP, // can't catch
	syscall.SIGURG,
	syscall.SIGUSR1,
}

func monitorSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 64)
	signal.Notify(ch, signals[:]...)
	for sig := range ch {
		if err := cmd.Process.Signal(sig); err != nil {
			// WARN: do something
		}
	}
}

func realMain(name string, args []string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return err // fast exit
	}
	opts := EnvOptions()
	_ = opts

	// stderr_r, stderr_w := io.Pipe()

	// scan := opts.Init(r, cmd, ...)
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Env = cleanEnv()
	rc, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	scan := opts.Init(rc, name)
	_ = scan

	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := cmd.Wait(); err != nil {
			Fatal(err) // WARN: don't actually do this
		}
	}()
	go func() {
		defer wg.Done()
		if err := scan.Consume(os.Stderr); err != nil {
			Fatal(err) // WARN: don't actually do this
		}
	}()

	wg.Wait()

	return nil
}

func xmain() {
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

func main() {
	{
		b := allocBytes(0, 1024*1024*1024*4)
		time.Sleep(time.Hour)
		fmt.Printf("%c", b[0])
		return
	}
	{
		buf := bytes.NewBuffer([]byte("hello!"))
		// r := allocBufioReader(buf, 32*1024*1024)
		r := bufio.NewReaderSize(buf, 32*1024*1024)
		_ = r
		time.Sleep(time.Second * 30)
		return
	}

	r := NewReader(strings.NewReader(TestInput))
	var buf bytes.Buffer
	if err := r.Copy(&buf); err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
}

// touchReader ensures a slice is fully allocated.
type touchReader struct{}

// Read touches each byte of p forcing it to be fully allocated.
func (touchReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func allocBufioReader(rd io.Reader, size int) *bufio.Reader {
	var scratch [8]byte
	b := bufio.NewReaderSize(touchReader{}, size)
	b.Read(scratch[:])
	b.Reset(rd)
	return b
}

func allocBytes(len, cap int) []byte {
	b := make([]byte, cap)
	for i := range b {
		b[i] = 0
	}
	return b[0:len:cap]
}

const TestInput = ``

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
