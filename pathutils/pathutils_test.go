package pathutils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"text/tabwriter"
	"time"
)

var _ = runtime.GOMAXPROCS(1)

type PathTest struct {
	path, result string
}

func (p PathTest) Path() []byte   { return []byte(p.path) }
func (p PathTest) Result() []byte { return []byte(p.result) }

var cleantests = []PathTest{
	// Already clean
	{"abc", "abc"},
	{"abc/def", "abc/def"},
	{"a/b/c", "a/b/c"},
	{".", "."},
	{"..", ".."},
	{"../..", "../.."},
	{"../../abc", "../../abc"},
	{"/abc", "/abc"},
	{"/", "/"},

	// Empty is current dir
	{"", "."},

	// Remove trailing slash
	{"abc/", "abc"},
	{"abc/def/", "abc/def"},
	{"a/b/c/", "a/b/c"},
	{"./", "."},
	{"../", ".."},
	{"../../", "../.."},
	{"/abc/", "/abc"},

	// Remove doubled slash
	{"abc//def//ghi", "abc/def/ghi"},
	{"//abc", "/abc"},
	{"///abc", "/abc"},
	{"//abc//", "/abc"},
	{"abc//", "abc"},

	// Remove . elements
	{"abc/./def", "abc/def"},
	{"/./abc/def", "/abc/def"},
	{"abc/.", "abc"},

	// Remove .. elements
	{"abc/def/ghi/../jkl", "abc/def/jkl"},
	{"abc/def/../ghi/../jkl", "abc/jkl"},
	{"abc/def/..", "abc"},
	{"abc/def/../..", "."},
	{"/abc/def/../..", "/"},
	{"abc/def/../../..", ".."},
	{"/abc/def/../../..", "/"},
	{"abc/def/../../../ghi/jkl/../../../mno", "../../mno"},
	{"/../abc", "/abc"},

	// Combinations
	{"abc/./../def", "def"},
	{"abc//./../def", "def"},
	{"abc/../../././../def", "../../def"},
}

var wincleantests = []PathTest{
	{`c:`, `c:.`},
	{`c:\`, `c:\`},
	{`c:\abc`, `c:\abc`},
	{`c:abc\..\..\.\.\..\def`, `c:..\..\def`},
	{`c:\abc\def\..\..`, `c:\`},
	{`c:\..\abc`, `c:\abc`},
	{`c:..\abc`, `c:..\abc`},
	{`\`, `\`},
	{`/`, `\`},
	{`\\i\..\c$`, `\c$`},
	{`\\i\..\i\c$`, `\i\c$`},
	{`\\i\..\I\c$`, `\I\c$`},
	{`\\host\share\foo\..\bar`, `\\host\share\bar`},
	{`//host/share/foo/../baz`, `\\host\share\baz`},
	{`\\a\b\..\c`, `\\a\b\c`},
	{`\\a\b`, `\\a\b`},
}

func TestClean(t *testing.T) {
	tests := cleantests
	if runtime.GOOS == "windows" {
		for i := range tests {
			tests[i].result = string(FromSlash(tests[i].Result()))
		}
		tests = append(tests, wincleantests...)
	}
	for _, test := range tests {
		if s := string(Clean(test.Path())); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.path, s, test.result)
		}
		if s := string(Clean(test.Result())); s != test.result {
			t.Errorf("Clean(%q) = %q, want %q", test.result, s, test.result)
		}
	}

	if testing.Short() {
		t.Skip("skipping malloc count in short mode")
	}
	if runtime.GOMAXPROCS(0) > 1 {
		t.Log("skipping AllocsPerRun checks; GOMAXPROCS>1")
		return
	}

	for _, test := range tests {
		allocs := testing.AllocsPerRun(100, func() { Clean(test.Result()) })
		if allocs > 0 {
			t.Errorf("Clean(%q): %v allocs, want zero", test.result, allocs)
		}
	}
}

type ExtTest struct {
	path, ext string
}

func (e ExtTest) Path() []byte { return []byte(e.path) }

var exttests = []ExtTest{
	{"path.go", ".go"},
	{"path.pb.go", ".go"},
	{"a.dir/b", ""},
	{"a.dir/b.go", ".go"},
	{"a.dir/", ""},
}

func TestExt(t *testing.T) {
	for _, test := range exttests {
		if x := string(Ext(test.Path())); x != test.ext {
			t.Errorf("Ext(%q) = %q, want %q", test.path, x, test.ext)
		}
	}
}

var basetests = []PathTest{
	{"", "."},
	{".", "."},
	{"/.", "."},
	{"/", "/"},
	{"////", "/"},
	{"x/", "x"},
	{"abc", "abc"},
	{"abc/def", "def"},
	{"a/b/.x", ".x"},
	{"a/b/c.", "c."},
	{"a/b/c.x", "c.x"},
}

var winbasetests = []PathTest{
	{`c:\`, `\`},
	{`c:.`, `.`},
	{`c:\a\b`, `b`},
	{`c:a\b`, `b`},
	{`c:a\b\c`, `c`},
	{`\\host\share\`, `\`},
	{`\\host\share\a`, `a`},
	{`\\host\share\a\b`, `b`},
}

func TestBase(t *testing.T) {
	tests := basetests
	if runtime.GOOS == "windows" {
		// make unix tests work on windows
		for i := range tests {
			tests[i].result = string(Clean(tests[i].Result()))
		}
		// add windows specific tests
		tests = append(tests, winbasetests...)
	}
	for _, test := range tests {
		if test.path == "" {
			continue // CEV: filepath returns "." but we return nil
		}
		if s := string(Base(test.Path())); s != test.result {
			t.Errorf("Base(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

var dirtests = []PathTest{
	{"", "."},
	{".", "."},
	{"/.", "/"},
	{"/", "/"},
	{"////", "/"},
	{"/foo", "/"},
	{"x/", "x"},
	{"abc", "."},
	{"abc/def", "abc"},
	{"a/b/.x", "a/b"},
	{"a/b/c.", "a/b"},
	{"a/b/c.x", "a/b"},
}

var windirtests = []PathTest{
	{`c:\`, `c:\`},
	{`c:.`, `c:.`},
	{`c:\a\b`, `c:\a`},
	{`c:a\b`, `c:a`},
	{`c:a\b\c`, `c:a\b`},
	{`\\host\share`, `\\host\share`},
	{`\\host\share\`, `\\host\share\`},
	{`\\host\share\a`, `\\host\share\`},
	{`\\host\share\a\b`, `\\host\share\a`},
}

func TestDir(t *testing.T) {
	tests := dirtests
	if runtime.GOOS == "windows" {
		// make unix tests work on windows
		for i := range tests {
			tests[i].result = string(Clean(tests[i].Result()))
		}
		// add windows specific tests
		tests = append(tests, windirtests...)
	}
	for _, test := range tests {
		if s := string(Dir(test.Path())); s != test.result {
			t.Errorf("Dir(%q) = %q, want %q", test.path, s, test.result)
		}
	}
}

func TestReader(t *testing.T) {
	const data = "foo\nbar\nbaz"
	exp := []string{"foo", "bar", "baz"}
	var got []string
	r := NewReader(bufio.NewReader(strings.NewReader(data)))
	for {
		b, err := r.ReadBytes('\n')
		if len(b) > 0 {
			got = append(got, string(b))
		}
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Lines: got: %q want: %q", got, exp)
	}
}

func diffLines(t *testing.T, a1, a2 []string) string {
	if len(a1) != len(a2) {
		t.Errorf("input lengths differ: a1: %d a2: %d", len(a1), len(a2))
	}

	n := len(a1)
	if len(a2) > n {
		n = len(a2)
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	for i := 0; i < n; i++ {
		switch {
		case i >= len(a1):
			fmt.Fprintf(w, "%d:\t%q\t%q\n", i, "<OOB>", a2[i])
		case i >= len(a2):
			fmt.Fprintf(w, "%d:\t%q\t%q\n", i, a1[i], "<OOB>")
		default:
			fmt.Fprintf(w, "%d:\t%q\t%q\n", i, a1[i], a2[i])
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}

func fuzzTestReader(t *testing.T, rr *rand.Rand) {
	const size = 32

	exp := make([]string, 32)
	for i := range exp {
		c := rune('a' + (i % ('z' - 'a')))
		exp[i] = strings.Repeat(string(c), rr.Intn(size*2)+1)
	}
	in := strings.Join(exp, "\n")
	r := NewReader(bufio.NewReaderSize(strings.NewReader(in), size))
	var got []string
	for {
		b, err := r.ReadBytes('\n')
		if len(b) > 0 {
			got = append(got, string(b))
		}
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Lines:\n%s\n", diffLines(t, got, exp))
	}
}

func TestReaderFuzz(t *testing.T) {
	numCPU := runtime.NumCPU()
	if numCPU >= 4 {
		numCPU /= 2
	} else {
		numCPU = 2
	}
	for i := 0; i < numCPU; i++ {
		t.Run(fmt.Sprintf("Fuzz_%d", i), func(t *testing.T) {
			t.Parallel()
			rr := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := 0; i < 100; i++ {
				fuzzTestReader(t, rr)
			}
		})
	}
}

func BenchmarkBase(b *testing.B) {
	name := []byte("drivers/staging/media/atomisp/pci/hive_isp_css_include/gp_timer.h")
	b.SetBytes(int64(len(name)))
	for i := 0; i < b.N; i++ {
		Base(name)
	}
}

func BenchmarkReader(b *testing.B) {
	rr := rand.New(rand.NewSource(123))

	var data []byte
	for i := 0; i < 4096; i++ {
		data = append(data, strings.Repeat("a", rr.Intn(8192))...)
		data = append(data, '\n')
	}
	rd := bytes.NewReader(data)

	br := bufio.NewReader(rd)
	r := NewReader(br)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rd.Seek(0, io.SeekStart)
		br.Reset(rd)

		for {
			_, err := r.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					b.Fatal(err)
				}
				break
			}
		}
	}
}
