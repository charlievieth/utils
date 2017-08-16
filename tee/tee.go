package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var Append bool

func init() {
	flag.BoolVar(&Append, "a", false, "Append the output to the files rather than overwriting them.")
}

type multiWriter struct {
	writers []io.Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

func realMain() error {
	flag.Parse()

	mode := os.O_CREATE | os.O_WRONLY
	if Append {
		mode |= os.O_APPEND
	} else {
		mode |= os.O_TRUNC
	}

	var w multiWriter
	w.writers = append(w.writers, os.Stdout)

	for _, name := range flag.Args() {
		f, err := os.OpenFile(name, mode, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		w.writers = append(w.writers, f)
	}

	if _, err := io.Copy(&w, os.Stdin); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
