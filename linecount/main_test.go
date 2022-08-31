package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/charlievieth/fastwalk"
)

func TestMain(m *testing.M) {
	fn := func() int {
		defer func() {
			benchData.once.Do(func() {})
			if benchData.temp != "" {
				os.RemoveAll(benchData.temp)
			}
		}()
		return m.Run()
	}
	os.Exit(fn())
}

func BenchmarkWalk(b *testing.B) {
	root := filepath.Join(runtime.GOROOT(), "src")
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		b.Skip("Benchmark requires Go source code")
	}
	b.ReportAllocs()
	w := &walker{
		exts:   make(map[string]int64),
		ignore: make(map[string]bool),
	}
	for i := 0; i < b.N; i++ {
		if err := fastwalk.Walk(nil, root, w.Walk); err != nil {
			b.Fatal(err)
		}
	}
}
