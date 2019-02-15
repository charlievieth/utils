package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
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
	r.buf = append(r.buf, frag...)
	// if len(r.buf) != 0 {
	// 	r.buf = r.buf[:len(r.buf)-1]
	// }
	return r.buf, err
}

func isNumber(b []byte) bool {
	for _, c := range b {
		// HERE HERE HERE HERE
		if c-'0' > '9' {
			return false
		}
		// if c < '0' || c > '9' {
		// 	return false
		// }
	}
	return true
}

func readHistory(f *os.File) error {
	r := Reader{
		b:   bufio.NewReader(f),
		buf: make([]byte, 128),
	}
	w := bufio.NewWriter(os.Stdout)
	var err error
	for {
		b, e := r.ReadBytes('\n')
		if len(b) != 0 && b[0] != '#' {
			if _, ew := w.Write(b); ew != nil {
				if e == nil || e == io.EOF {
					e = ew
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
	return err
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func homedir() (string, error) {
	if s := os.Getenv("HOME"); s != "" && isDir(s) {
		return s, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func realMain() error {
	home, err := homedir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".bash_history")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return readHistory(f)
}

func isDigit(c byte) bool {
	return c-'0' <= 9
}

func main() {
	{
		for i := '!'; i < '~'; i++ {
			fmt.Printf("%c: %t\n", i, isDigit(byte(i)))
		}
		return
	}

	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
