package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TODO:
//  * Enforce max line length
//  * Use a file for intermediate panic storage, this allows
//    us to handle very large panics without allocating
//  * Make all this configurable (including file location)

const (
	panicPrefix = "panic: "

	// use '\r\n' as the default newline replacement since it
	// is more distinctive than just '\n'
	defaultReplacement = "\\r\\n"

	defaultMaxLineLength = 16 * 1014

	// Use a buffer large enough to capture large stack traces in one read.
	// This should help in OOM situations by allowing the crashing program
	// to dump its stack and exit before we format the message.
	defaultReaderSize = 16 * 1024 * 1024
)

var panicBytes = []byte(panicPrefix)

// TODO: reserve space in case we are in an OOM situation
type Reader struct {
	b         *bufio.Reader
	buf       []byte // line buffer
	trace     []byte // trace buffer
	panicking bool
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		b:     bufio.NewReaderSize(r, defaultReaderSize),
		buf:   make([]byte, 0, 256),
		trace: make([]byte, 0, 32*1024),
	}
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
	return ReplaceNewLines(trace, []byte(defaultReplacement), nil)
}

// TODO:
// * cap line length
// * consume all of the panic
func (r *Reader) ReadBytes(delim byte) ([]byte, error) {
	var frag []byte
	var err error
	r.buf = r.buf[:0]
	for {
		var e error
		frag, e = r.b.ReadSlice(delim)

		// the panic check must occur before breaking the loop
		if !r.panicking && len(frag) >= len(panicPrefix) {
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

// CEV: assume stderr for now
func (r *Reader) Copy(w io.Writer) (err error) {
	for {
		b, er := r.ReadBytes('\n')
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

func realMain(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout

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

const TestInput = `
log line 1
log line 2
log line 3
log line 4
log line 5
log line 6

panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x40 pc=0x7260d4]

goroutine 5 [running]:
golang.org/x/tools/internal/lsp/source.qualifier(0x0, 0x0, 0x0, 0x8fa280)
	/home/nohup/Golang/src/golang.org/x/tools/internal/lsp/source/completion_format.go:173 +0x34
golang.org/x/tools/internal/lsp/source.DocumentSymbols(0x8f5420, 0xc000070fc0, 0x8f93a0, 0xc000237170, 0xc0001893e0, 0x2d, 0x8f93a0)
	/home/nohup/Golang/src/golang.org/x/tools/internal/lsp/source/symbols.go:48 +0x151
golang.org/x/tools/internal/lsp.(*Server).documentSymbol(0xc00014a4d0, 0x8f5420, 0xc000070fc0, 0xc00017fbf0, 0xc00017fbf0, 0x0, 0x0, 0x0, 0xc000168320)
	/home/nohup/Golang/src/golang.org/x/tools/internal/lsp/symbols.go:22 +0x138
golang.org/x/tools/internal/lsp.(*Server).DocumentSymbol(0xc00014a4d0, 0x8f5420, 0xc000070fc0, 0xc00017fbf0, 0xc00017fbf0, 0x0, 0x0, 0x0, 0x0)
	/home/nohup/Golang/src/golang.org/x/tools/internal/lsp/server.go:198 +0x4d
golang.org/x/tools/internal/lsp/protocol.serverHandler.func1(0x8f5420, 0xc000070fc0, 0xc00014a540, 0xc00000ef80)
	/home/nohup/Golang/src/golang.org/x/tools/internal/lsp/protocol/tsserver.go:346 +0x4adb
golang.org/x/tools/internal/jsonrpc2.(*Conn).Run.func1(0xc000026ba0, 0xc00014a540)
	/home/nohup/Golang/src/golang.org/x/tools/internal/jsonrpc2/jsonrpc2.go:276 +0xda
created by golang.org/x/tools/internal/jsonrpc2.(*Conn).Run
	/home/nohup/Golang/src/golang.org/x/tools/internal/jsonrpc2/jsonrpc2.go:270 +0xba
[Error - 3:27:07 AM] Connection to server got closed. Server will not be restarted.
[Error - 3:27:07 AM] Request textDocument/documentSymbol failed.
Error: Connection got disposed.
    at Object.dispose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/main.js:876:25)
    at Object.dispose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:57:35)
    at LanguageClient.handleConnectionClosed (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:2036:42)
    at LanguageClient.handleConnectionClosed (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/main.js:127:15)
    at closeHandler (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:2023:18)
    at CallbackList.invoke (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:62:39)
    at Emitter.fire (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:120:36)
    at closeHandler (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/main.js:226:26)
    at CallbackList.invoke (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:62:39)
    at Emitter.fire (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:120:36)
    at StreamMessageReader.fireClose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/messageReader.js:111:27)
    at Socket.listen.readable.on (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/messageReader.js:151:46)
    at Socket.emit (events.js:187:15)
    at Pipe.Socket._destroy._handle.close [as _onclose] (net.js:596:12)
[Error - 3:27:07 AM] Request textDocument/codeAction failed.
Error: Connection got disposed.
    at Object.dispose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/main.js:876:25)
    at Object.dispose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:57:35)
    at LanguageClient.handleConnectionClosed (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:2036:42)
    at LanguageClient.handleConnectionClosed (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/main.js:127:15)
    at closeHandler (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:2023:18)
    at CallbackList.invoke (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:62:39)
    at Emitter.fire (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:120:36)
    at closeHandler (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/main.js:226:26)
    at CallbackList.invoke (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:62:39)
    at Emitter.fire (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:120:36)
    at StreamMessageReader.fireClose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/messageReader.js:111:27)
    at Socket.listen.readable.on (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/messageReader.js:151:46)
    at Socket.emit (events.js:187:15)
    at Pipe.Socket._destroy._handle.close [as _onclose] (net.js:596:12)
[Error - 3:27:07 AM] Request textDocument/documentLink failed.
Error: Connection got disposed.
    at Object.dispose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/main.js:876:25)
    at Object.dispose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:57:35)
    at LanguageClient.handleConnectionClosed (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:2036:42)
    at LanguageClient.handleConnectionClosed (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/main.js:127:15)
    at closeHandler (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-languageclient/lib/client.js:2023:18)
    at CallbackList.invoke (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:62:39)
    at Emitter.fire (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:120:36)
    at closeHandler (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/main.js:226:26)
    at CallbackList.invoke (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:62:39)
    at Emitter.fire (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/events.js:120:36)
    at StreamMessageReader.fireClose (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/messageReader.js:111:27)
    at Socket.listen.readable.on (/home/nohup/.vscode-insiders/extensions/ms-vscode.go-0.10.2/node_modules/vscode-jsonrpc/lib/messageReader.js:151:46)
    at Socket.emit (events.js:187:15)
    at Pipe.Socket._destroy._handle.close [as _onclose] (net.js:596:12)
`
