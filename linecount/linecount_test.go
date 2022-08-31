package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	crand "crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func createTestFile(t testing.TB, lines int64, prefix, suffix string) string {
	name := filepath.Join(t.TempDir(), fmt.Sprintf("%d.txt", lines))
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 32*1024)
	if prefix != "" {
		w.WriteString(prefix)
		w.WriteByte(' ')
	}

	buf := make([]byte, 0, 32)
	for i := int64(0); i < lines; i++ {
		w.Write(strconv.AppendInt(buf[:0], i, 10))
		if suffix != "" {
			w.WriteByte(' ')
			w.WriteString(suffix)
		}
		if err := w.WriteByte('\n'); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestLineCount(t *testing.T) {
	for _, lines := range []int64{1, 100, 10_000, 100_000} {
		t.Run(fmt.Sprint(lines), func(t *testing.T) {
			name := createTestFile(t, lines, "", "")
			check := func(n int64, ext string, err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
				if n != lines {
					t.Errorf("lines: got: %d want: %d", n, lines)
				}
				if ext != "" {
					t.Errorf("ext: got: %q want: %q", ext, "")
				}
			}
			check(lineCount(name, false))
		})
	}
}

func TestLineCountNeedExt(t *testing.T) {
	const lines = 100
	expExt := "sh"

	name := createTestFile(t, lines, "#!/bin/sh", "")
	check := func(n int64, ext string, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if n != lines {
			t.Errorf("lines: got: %d want: %d", n, lines)
		}
		if ext != expExt {
			t.Errorf("ext: got: %q want: %q", ext, expExt)
		}
	}
	check(lineCount(name, true))

	expExt = ""
	check(lineCount(name, false))
}

func TestLineCountBinary(t *testing.T) {
	name := filepath.Join(t.TempDir(), "test.bin")
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, maxBinaryReadSize)

	test := func(t *testing.T) (binary bool) {
		f, err := os.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		if _, err := crand.Reader.Read(buf); err != nil {
			t.Fatal(err)
		}
		binary = bytes.Contains(buf, []byte{0})

		if _, err := f.Write(buf); err != nil {
			t.Fatal(err)
		}

		var exp error
		if binary {
			exp = ErrBinary
		}
		if _, _, err := lineCount(name, false); err != exp {
			t.Errorf("binary: %t expected error: %v got: %v", binary, exp, err)
		}
		return binary
	}

	for n, i := 0, 0; n < 5; i++ {
		if test(t) {
			n++
		}
		if i > 1_000 {
			t.Fatalf("failed to create %d binary files after %d runs", 5, i)
		}
	}
}

type infiniteReader struct {
	*bytes.Reader
}

func (r *infiniteReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		err = nil
		r.Reader.Seek(0, 0)
	}
	return n, err
}

func TestInfiniteReader(t *testing.T) {
	r := infiniteReader{bytes.NewReader([]byte("ab"))}
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, &r, 1023); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 1023 {
		t.Errorf("want: %d got: %d", 1023, buf.Len())
	}
}

var benchData struct {
	once sync.Once
	temp string // temp dir
	data []byte
	err  error
}

func loadBenchDataOnce(t testing.TB) (string, []byte) {
	d := &benchData
	benchData.once.Do(func() {
		d.temp, d.err = os.MkdirTemp("", "linecount-*")
		if d.err != nil {
			t.Fatal(d.err)
			return
		}
		f, err := os.Open("testdata/256K.go.data.gz")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		r, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		d.data, d.err = io.ReadAll(r)
		if d.err != nil {
			t.Fatal(d.err)
		}
		if d.err = r.Close(); d.err != nil {
			t.Fatal(d.err)
		}
	})
	if d.err != nil {
		t.Fatal(d.err)
	}
	return d.temp, d.data
}

var benchFilesCache sync.Map

func createBenchFile(b *testing.B, size int64) string {
	if size <= 0 {
		b.Fatal("non-positive file size:", size)
	}

	type cachedFile struct {
		once sync.Once
		name string
	}
	v, ok := benchFilesCache.Load(size)
	if !ok {
		v, _ = benchFilesCache.LoadOrStore(size, new(cachedFile))
	}
	cf := v.(*cachedFile)

	// only create the file once
	cf.once.Do(func() {
		temp, data := loadBenchDataOnce(b)

		name := filepath.Join(temp, fmt.Sprintf("%d.go.test", size))

		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			b.Fatal(err)
		}
		defer f.Close()

		r := infiniteReader{bytes.NewReader(data)}
		if _, err := io.CopyN(f, &r, size); err != nil {
			b.Fatal(err)
		}
		if err := f.Close(); err != nil {
			b.Fatal(err)
		}
		cf.name = name
	})

	// Name is only empty if an error occured
	if cf.name == "" {
		b.Fatalf("failed to create bench file with size %d due to a previous error", size)
	}
	return cf.name
}

func warmupFile(b *testing.B, name string) {
	fn := func(name string) error {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		if _, err := io.Copy(io.Discard, f); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	}
	for i := 0; i < 20; i++ {
		if err := fn(name); err != nil {
			b.Fatal(err)
		}
	}
}

type lineCountFunc func(name string, needExt bool) (lines int64, ext string, err error)

func benchmarkLineCount(b *testing.B, size int64, fn lineCountFunc) {
	name := createBenchFile(b, size)
	warmupFile(b, name)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := fn(name, false); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkLineCountParallel(b *testing.B, size int64, fn lineCountFunc) {
	name := createBenchFile(b, size)
	warmupFile(b, name)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, _, err := fn(name, false); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// func BenchmarkLineCountXXX(b *testing.B) {
// 	const (
// 		kB = 1024
// 		mB = 1024 * 1024
// 		gB = 1024 * 1024 * 1024
// 	)
// 	sizes := []int64{
// 		// 4096,
// 		// 8192,
// 		// 32 * kB,
// 		// 64 * kB,
// 		128 * kB,
// 		mB,
// 		4 * mB,
// 		16 * mB,
// 		32 * mB,
// 		64 * mB,
// 	}
// 	for _, size := range sizes {
// 		kb := size / kB
// 		b.Run(fmt.Sprintf("LineCountFile/%dKb", kb), func(b *testing.B) {
// 			benchmarkLineCountParallel(b, size, LineCount)
// 		})
// 	}
// }

func BenchmarkLineCountFile(b *testing.B) {
	const lines = 100_000
	name := createTestFile(b, lines, "#!/bin/sh", strings.Repeat("a", 128))

	f, err := os.Open(name)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	fi, err := os.Stat(name)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Lines: %d Bytes: %d", lines, fi.Size())
	b.SetBytes(fi.Size())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := f.Seek(0, 0); err != nil {
			b.Fatal(err)
		}
		lineCountFile(f, false)
	}
}
