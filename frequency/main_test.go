package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"io"
	"os"
	"slices"
	"sync"
	"testing"
)

const (
	Dotfiles    = "testdata/dotfiles.txt.gz"
	ExpDotfiles = "testdata/dotfiles.exp.gob.gz"
	Gofiles     = "testdata/go.txt.gz"
	ExpGofiles  = "testdata/go.exp.gob.gz"
)

var (
	decodeCache sync.Map
	gunzipCache sync.Map
)

func decode(t testing.TB, name string) []Line {
	if v, ok := decodeCache.Load(name); ok {
		return *(v.(*[]Line))
	}
	lines, err := doDecode(name)
	if err != nil {
		t.Fatal(err)
	}
	decodeCache.Store(name, &lines)
	return lines
}

func gunzip(t testing.TB, name string) []byte {
	if v, ok := decodeCache.Load(name); ok {
		return v.([]byte)
	}
	data, err := doGunzip(name)
	if err != nil {
		t.Fatal(err)
	}
	decodeCache.Store(name, data)
	return data
}

func doGunzip(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func doDecode(name string) ([]Line, error) {
	data, err := doGunzip(name)
	if err != nil {
		return nil, err
	}
	var lines []Line
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&lines); err != nil {
		return nil, err
	}
	return lines, nil
}

func testReadline(t *testing.T, dataFile, expFile string) {
	data := gunzip(t, dataFile)
	exp := decode(t, expFile)
	rd := bytes.NewReader(data)
	r := Reader{
		b:   bufio.NewReader(rd),
		buf: make([]byte, 128),
	}
	lines, err := ReadLines(&r, '\n', false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(lines, exp) {
		t.Error(dataFile)
	}
}

func TestReadline(t *testing.T) {
	testReadline(t, Dotfiles, ExpDotfiles)
}

func TestReadline_Hard(t *testing.T) {
	testReadline(t, Gofiles, ExpGofiles)
}

func benchmarkReadLines(b *testing.B, name string, ignoreCase bool) {
	data := gunzip(b, name)
	rd := bytes.NewReader(data)
	r := Reader{
		b:   bufio.NewReader(rd),
		buf: make([]byte, 128),
	}
	if _, err := ReadLines(&r, '\n', ignoreCase, false, false); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rd.Seek(0, 0)
		if _, err := ReadLines(&r, '\n', ignoreCase, false, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadLines_Short(b *testing.B) {
	benchmarkReadLines(b, Dotfiles, false)
}

func BenchmarkReadLines_Short_Case(b *testing.B) {
	benchmarkReadLines(b, Dotfiles, true)
}

func BenchmarkReadLines_Long(b *testing.B) {
	benchmarkReadLines(b, Gofiles, false)
}

func BenchmarkReadLines_Long_Case(b *testing.B) {
	benchmarkReadLines(b, Gofiles, true)
}
