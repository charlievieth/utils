package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"
)

var benchData []byte

func initBenchData(t testing.TB) {
	if benchData != nil {
		return
	}
	f, err := os.Open("testdata/words.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	if err := gr.Close(); err != nil {
		t.Fatal(err)
	}
	benchData = data
}

func BenchmarkStreamLines(b *testing.B) {
	initBenchData(b)
	r := bytes.NewReader(benchData)
	b.SetBytes(int64(len(benchData)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		StreamLines(r, io.Discard, '\n', false)
	}
}
