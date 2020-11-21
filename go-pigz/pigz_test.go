package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

const Gettysburg = "  Four score and seven years ago our fathers brought forth on\n" +
	"this continent, a new nation, conceived in Liberty, and dedicated\n" +
	"to the proposition that all men are created equal.\n" +
	"  Now we are engaged in a great Civil War, testing whether that\n" +
	"nation, or any nation so conceived and so dedicated, can long\n" +
	"endure.\n" +
	"  We are met on a great battle-field of that war.\n" +
	"  We have come to dedicate a portion of that field, as a final\n" +
	"resting place for those who here gave their lives that that\n" +
	"nation might live.  It is altogether fitting and proper that\n" +
	"we should do this.\n" +
	"  But, in a larger sense, we can not dedicate — we can not\n" +
	"consecrate — we can not hallow — this ground.\n" +
	"  The brave men, living and dead, who struggled here, have\n" +
	"consecrated it, far above our poor power to add or detract.\n" +
	"The world will little note, nor long remember what we say here,\n" +
	"but it can never forget what they did here.\n" +
	"  It is for us the living, rather, to be dedicated here to the\n" +
	"unfinished work which they who fought here have thus far so\n" +
	"nobly advanced.  It is rather for us to be here dedicated to\n" +
	"the great task remaining before us — that from these honored\n" +
	"dead we take increased devotion to that cause for which they\n" +
	"gave the last full measure of devotion —\n" +
	"  that we here highly resolve that these dead shall not have\n" +
	"died in vain — that this nation, under God, shall have a new\n" +
	"birth of freedom — and that government of the people, by the\n" +
	"people, for the people, shall not perish from this earth.\n" +
	"\n" +
	"Abraham Lincoln, November 19, 1863, Gettysburg, Pennsylvania\n"

func CompressReference(t *testing.T, data []byte, level int) ([]byte, string) {
	dst := new(bytes.Buffer)
	pg, err := NewWriterLevel(dst, level)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pg.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := pg.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Grow(len(data) + bytes.MinRead)
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != len(data) {
		t.Errorf("Decompress length: got: %d want: %d", buf.Len(), len(data))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		i := 0
		b := buf.Bytes()
		for ; i < len(b) && b[i] == data[i]; i++ {
		}
		t.Errorf("Decompress bytes not equal starting at offset: %d", i)
	}
	return nil, ""
}

const b32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

func TestCompressor(t *testing.T) {
	tests := []string{
		"",
		"hello world\n",
		"hello world\n" +
			"hello world\n",
		"she sells seashells by the seashore\n",
		Gettysburg,
		strings.Repeat(b32, 1024*1024*32/len(b32)),
	}
	for i, str := range tests {
		t.Logf("Test #%d", i)
		CompressReference(t, []byte(str), DefaultCompression)
	}
}

func TestLevels(t *testing.T) {
	for i := NoCompression; i <= BestCompression; i++ {
		level := i
		t.Run(fmt.Sprintf("%d", level), func(t *testing.T) {
			CompressReference(t, []byte(Gettysburg), level)
		})
	}
}

// func bench(b *testing.B, in *bytes.Reader, level int) {
// 	pg, err := NewWriterLevel(ioutil.Discard, level)
// 	if err != nil {
// 		b.Fatal(err)
// 	}
// 	in.Seek(0, 0)
// 	// pg.
// }

var benchData = bytes.Repeat([]byte(b32), 1024*1024*16/len(b32))

func BenchmarkCompressor_Large(b *testing.B) {
	// data := bytes.Repeat([]byte(b32), 1024*1024*16/len(b32))
	rd := bytes.NewReader(benchData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pg, err := NewWriterLevel(ioutil.Discard, DefaultCompression)
		if err != nil {
			b.Fatal(err)
		}
		rd.Seek(0, 0)
		io.Copy(pg, rd)
	}
}

func BenchmarkCompressor_Large_Reference(b *testing.B) {
	data := bytes.Repeat([]byte(b32), 1024*1024*16/len(b32))
	rd := bytes.NewReader(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gz, err := gzip.NewWriterLevel(ioutil.Discard, DefaultCompression)
		if err != nil {
			b.Fatal(err)
		}
		rd.Seek(0, 0)
		io.Copy(gz, rd)
	}
}
