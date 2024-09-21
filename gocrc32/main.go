package main

import (
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"slices"
)

const BlockSize = 256 * 1024

func checksum(filename string, buf []byte) (uint32, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if cap(buf) < BlockSize {
		buf = slices.Grow(buf, BlockSize)
	}
	h := crc32.NewIEEE()
	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)
	flag.Parse()
	buf := make([]byte, BlockSize)
	for _, name := range flag.Args() {
		u, err := checksum(name, buf)
		if err != nil {
			log.Printf("%s: %v\n", name, err)
			continue
		}
		fmt.Printf("%08X\t%s\n", u, name)
	}
}
