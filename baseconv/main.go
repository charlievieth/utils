package main

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"
)

func Parse(s string) {
	if len(s) == 0 {
		return
	}
	x, ok := new(big.Int).SetString(s, 0)
	if !ok {
		fmt.Fprintf(os.Stderr, "error parsing: %s\n", s)
		return
	}
	fmt.Printf("%s:\n", s)
	fmt.Printf("   8: %o\n", x)
	fmt.Printf("  10: %v\n", x)
	fmt.Printf("  16: %X\n", x)
}

func ReadStream() {
	const Timeout = time.Second

	done := make(chan error)
	var in bytes.Buffer
	go func() {
		_, err := in.ReadFrom(os.Stdin)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Fatal(err)
		}
	case <-time.After(Timeout):
		log.Fatal("timed out after:", Timeout)
	}

	for _, b := range bytes.Fields(in.Bytes()) {
		Parse(string(b))
	}
}

func main() {
	if len(os.Args) < 2 {
		ReadStream()
		return
	}
	for _, s := range os.Args[1:] {
		Parse(s)
	}
}
