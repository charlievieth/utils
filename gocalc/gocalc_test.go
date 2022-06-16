package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

type parseValueTest struct {
	in   string
	want *big.Float
	err  error
}

func mustParseFloat(s string) *big.Float {
	f, _, err := big.ParseFloat(s, 10, 200, 0)
	if err != nil {
		panic(err)
	}
	return f
}

func mul(x, y *big.Float) *big.Float {
	x.SetPrec(200)
	y.SetPrec(200)
	return new(big.Float).Mul(x, y)
}

var parseValueTests = []parseValueTest{
	{"0", big.NewFloat(0), nil},
	{"0K", big.NewFloat(0), nil},
	{"1K", big.NewFloat(1024), nil},
	{"1Kb", big.NewFloat(1024), nil},
	{"2kb", big.NewFloat(1024 * 2), nil},
	{"1.K", big.NewFloat(1024), nil},
	{"1.1K", mul(mustParseFloat("1.1"), big.NewFloat(1024)), nil},
	{"1234.5K", big.NewFloat(1234.5 * 1024), nil},
	{"1234.5E", big.NewFloat(1234.5 * (1 << (10 * 6))), nil},
	{"1234.5E", mul(big.NewFloat(1234.5), new(big.Float).SetUint64((1 << (10 * 6)))), nil},
	{"1234.56789E", mul(mustParseFloat("1234.56789"), new(big.Float).SetUint64((1 << (10 * 6)))), nil},
	{"1a", big.NewFloat(0), &SuffixError{Suffix: "a"}},
}

func TestParseValue(t *testing.T) {
	for _, test := range parseValueTests {
		got, err := ParseValue(test.in, true)
		if got == nil {
			got = new(big.Float)
		}
		if test.want.Cmp(got) != 0 || !reflect.DeepEqual(err, test.err) {
			t.Errorf("ParseValue(%q, %t) = %f, %v; want: %f, %v",
				test.in, true, got, err, test.want, test.err)
		}
	}
}

func TestProcess(t *testing.T) {
	buf := new(bytes.Buffer)
	wantSum := new(big.Float).SetPrec(200)
	for i := 0; i < 256; i++ {
		fmt.Fprintln(buf, uint64(math.MaxUint))
		fmt.Fprintln(buf, uint64(math.MaxInt))
		var x big.Float
		x.SetInt64(math.MaxInt64)
		wantSum.Add(wantSum, &x)
		x.SetUint64(math.MaxUint64)
		wantSum.Add(wantSum, &x)
	}
	var conf Config
	res, err := conf.Process(buf)
	if err != nil {
		t.Fatal(err)
	}
	if res.Sum.Cmp(wantSum) != 0 {
		t.Errorf("Sum: got: %f want: %f", res.Sum, wantSum)
	}
}

var files map[string][]byte

func readFile(t testing.TB, name string) []byte {
	if data, ok := files[name]; ok {
		return data
	}
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	rr := rand.New(rand.NewSource(1))
	lines := bytes.Split(b, []byte{'\n'})
	rr.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})
	data := bytes.Join(lines, []byte{'\n'})
	if files == nil {
		files = make(map[string][]byte)
	}
	files[name] = data
	return data
}

func benchmarkProcess(b *testing.B, human bool, filename string) {
	data := readFile(b, filename)
	conf := Config{Human: human}
	b.SetBytes(int64(len(data)))
	var r bytes.Reader
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Reset(data)
		if _, err := conf.Process(&r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcess(b *testing.B) {
	b.Run("Bytes", func(b *testing.B) {
		benchmarkProcess(b, false, "testdata/byte_sizes.txt.gz")
	})
	b.Run("Human", func(b *testing.B) {
		benchmarkProcess(b, true, "testdata/human_sizes.txt.gz")
	})
}
