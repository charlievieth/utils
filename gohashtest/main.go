package main

import (
	"bytes"
	"fmt"
	"hash"
	"hash/crc32"
	"log"
	"os"

	"github.com/cespare/xxhash"
	"github.com/twmb/murmur3"
)

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func forEachWord(fn func(b []byte)) {
	data, err := os.ReadFile("/usr/share/dict/web2")
	if err != nil {
		log.Fatal(err)
	}
	for _, word := range bytes.Split(data, []byte{'\n'}) {
		word = bytes.TrimSpace(word)
		if len(word) > 0 {
			fn(word)
		}
	}
}

func TestHashFunction64(name string, fn hash.Hash64) {
	colls := make(map[uint64]int)
	words := make(map[uint64][]string)
	forEachWord(func(word []byte) {
		fn.Reset()
		fn.Write(word)
		h := fn.Sum64()
		colls[h]++
		words[h] = append(words[h], string(word))
	})
	collision := false
	fmt.Printf("%s:\n", name)
	for h, n := range colls {
		if n > 1 {
			fmt.Printf("  %d: %s\n", n, words[h])
			collision = true
		}
	}
	if !collision {
		fmt.Println("  no collisions")
	}
}

type Hash64Adapter struct {
	hash.Hash32
}

func (h *Hash64Adapter) Sum64() uint64 {
	return uint64(h.Sum32())
}

func main() {
	TestHashFunction64("crc32", &Hash64Adapter{crc32.New(crc32.MakeTable(crc32.Castagnoli))})
	TestHashFunction64("xxhash", xxhash.New())
	TestHashFunction64("murmur3", murmur3.New64())
}
