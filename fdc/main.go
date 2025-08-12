package main

import (
	"fmt"
	"io/fs"
	"os"
	"sync/atomic"

	"github.com/charlievieth/fastwalk"
)

func main() {
	count := new(atomic.Int64)
	err := fastwalk.Walk(nil, ".", func(_ string, _ fs.DirEntry, err error) error {
		count.Add(1)
		return err
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(count.Load())
}
