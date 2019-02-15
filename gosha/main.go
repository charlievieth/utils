package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func HashFile(name string, h hash.Hash) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h.Reset()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func HashFileBuf(name string, h hash.Hash, buf []byte) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h.Reset()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func PrintHash(path string, sum []byte) error {
	if PrintBasename {
		path = filepath.Base(path)
	}
	_, err := fmt.Printf("%x  %s\n", sum, path)
	return err
}

func ReadStdinP(delim byte, basename bool) error {
	names := make(chan string, 32)
	stop := make(chan struct{})
	var first error
	var wg sync.WaitGroup
	var once sync.Once
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h := sha256.New()
			buf := make([]byte, 32*1024)
		Loop:
			for name := range names {
				select {
				case <-stop:
					return
				default:
					// ok
				}
				sum, err := HashFileBuf(name, h, buf)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error (%s): %s\n", name, err)
					continue Loop
				}
				n := hex.Encode(buf, sum)
				b := buf[:n]
				b = append(b, "  "...)
				if basename {
					name = filepath.Base(name)
				}
				b = append(b, name...)
				b = append(b, '\n')
				if _, err := os.Stdout.Write(b); err != nil {
					once.Do(func() {
						first = err
						close(stop)
					})
					return
				}
			}
		}()
	}
	r := bufio.NewReader(os.Stdin)
	var err error
Loop:
	for {
		select {
		case <-stop:
			fmt.Fprintln(os.Stderr, "WARN: STOP")
			break Loop
		default:
			b, er := r.ReadBytes(delim)
			if len(b) > 1 {
				name := string(b[:len(b)-1])
				names <- name
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break Loop
			}
		}
	}
	close(names)
	if err != nil {
		close(stop)
		wg.Wait()
		return err
	}
	close(stop)
	wg.Wait()
	return first
}

func ReadStdin(delim byte) error {
	r := bufio.NewReader(os.Stdin)
	h := sha256.New()
	var err error
	for {
		b, er := r.ReadBytes(delim)
		if len(b) > 1 {
			name := string(b[:len(b)-1])
			sum, eh := HashFile(name, h)
			if eh != nil {
				fmt.Fprintf(os.Stderr, "error (%s): %s\n", name, eh)
			} else {
				if ew := PrintHash(name, sum); ew != nil {
					err = ew
					break
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
	return err
}

func ReadArgs(args []string) error {
	h := sha256.New()
	for _, name := range args {
		b, err := HashFile(name, h)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error (%s): %s\n", name, err)
			continue
		}
		if _, err := fmt.Fprintf(os.Stdout, "%x  %s\n", b, name); err != nil {
			return err
		}
	}
	return nil
}

var (
	NullTerminated bool
	PrintBasename  bool
)

func init() {
	flag.BoolVar(&PrintBasename, "b", false, "Print file base name")
	flag.BoolVar(&NullTerminated, "0", false, "Stdin is null terminated")
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 || flag.Arg(0) == "-" {
		if flag.NArg() > 1 {
			Fatalf("extra args after '-'")
		}
		var delim byte
		if !NullTerminated {
			delim = '\n'
		}
		if err := ReadStdinP(delim, PrintBasename); err != nil {
			Fatalf("reading from stdin: %s", err)
		}
	}
	if err := ReadArgs(flag.Args()); err != nil {
		Fatal(err)
	}
}

func Fatalf(format string, a ...interface{}) {
	Fatal(fmt.Sprintf(format, a...))
}

func Fatal(err interface{}) {
	var s string
	if _, file, line, ok := runtime.Caller(1); ok {
		s = fmt.Sprintf("%s:%d", file, line)
	}
	if err != nil {
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
