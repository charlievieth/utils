package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"io"
	"os"
	"reflect"
	"testing"
)

var (
	Dotfiles    []byte
	ExpDotfiles []Line
	Gofiles     []byte
	ExpGofiles  []Line
)

func init() {
	var err error
	Dotfiles, err = gunzip("testdata/dotfiles.txt.gz")
	if err != nil {
		Fatal(err)
	}
	ExpDotfiles, err = decode("testdata/dotfiles.exp.gob.gz")
	if err != nil {
		Fatal(err)
	}
	Gofiles, err = gunzip("testdata/go.txt.gz")
	if err != nil {
		Fatal(err)
	}
	ExpGofiles, err = decode("testdata/go.exp.gob.gz")
	if err != nil {
		Fatal(err)
	}
}

func TestReadline(t *testing.T) {
	rd := bytes.NewReader(Dotfiles)
	r := Reader{
		b:   bufio.NewReader(rd),
		buf: make([]byte, 128),
	}
	lines, err := ReadLines(&r, '\n', false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(lines, ExpDotfiles) {
		t.Error("Error: Dotfiles")
	}
}

func TestReadline_Hard(t *testing.T) {
	rd := bytes.NewReader(Gofiles)
	r := Reader{
		b:   bufio.NewReader(rd),
		buf: make([]byte, 128),
	}
	lines, err := ReadLines(&r, '\n', false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(lines, ExpGofiles) {
		t.Error("Error: ExpGofiles")
	}
}

func benchmarkReadLines(b *testing.B, data []byte, ignoreCase bool) {
	rd := bytes.NewReader(data)
	r := Reader{
		b:   bufio.NewReader(rd),
		buf: make([]byte, 128),
	}
	if _, err := ReadLines(&r, '\n', ignoreCase); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rd.Seek(0, 0)
		if _, err := ReadLines(&r, '\n', ignoreCase); err != nil {
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

func decode(name string) ([]Line, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	var lines []Line
	if err := gob.NewDecoder(r).Decode(&lines); err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	return lines, nil
}

func gunzip(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
