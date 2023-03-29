package glob

import (
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

type matchTest struct {
	pattern  string
	basename string
	match    bool
}

var matchTests = []matchTest{
	{"*.c", "m.c", true},
	{"*.c", "m.h", false},
	{"m.*", "m.c", true},
	{"m.*", "x.c", false},
	{"m.c", "m.c", true},
	{"m.[ch]", "m.c", true},
	{"*main*", "main.c", true},
	{"a/b/*.c", "a/b/m.c", true},
	{"a/**/*.c", "a/b/m.c", true},
	{"a/**/*", "a/b/m.c", true},
	{"a/*b*/**", "a/b/m.c", true},
	{"*.c", "a/b/m.c", true},
	{"m.*", "a/b/m.c", true},
	{"*m.c*", "a/b/m.c", true},
	{"m.c", "a/b/m.c", true},

	// Negate tests
	{"!*.c", "m.c", false},
	{"!*.c", "m.h", true},
	{"!m.*", "m.c", false},
	{"!m.*", "x.c", true},
	{"!m.c", "m.c", false},
	{"!m.[ch]", "m.c", false},
	{"!*main*", "main.c", false},
	{"!*", "main.c", false},
	{"!a/b/*.c", "a/b/m.c", false},
}

func TestMatch(t *testing.T) {
	for _, x := range matchTests {
		g, err := Parse(x.pattern)
		if err != nil {
			t.Fatal(err)
		}
		got := g.Match(x.basename)
		if got != x.match {
			t.Errorf("%+v: %s: Match(%q) = %t; want: %t", x, g, x.basename, got, x.match)
		}
	}
}

func TestSet(t *testing.T) {
	for _, x := range matchTests {
		g, err := Parse("*a*")
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Set(x.pattern); err != nil {
			t.Fatal(err)
		}
		got := g.Match(x.basename)
		if got != x.match {
			t.Errorf("%+v: %s: Match(%q) = %t; want: %t", x, g, x.basename, got, x.match)
		}
	}
}

func TestPatter(t *testing.T) {
	for _, x := range matchTests {
		g, err := Parse(x.pattern)
		if err != nil {
			t.Fatal(err)
		}
		got := g.Pattern()
		if got != x.pattern {
			t.Errorf("%+v: Pattern() = `%s`; want: `%s`", g, got, x.pattern)
		}
		got = g.String()
		if got != x.pattern {
			t.Errorf("%+v: String() = `%s`; want: `%s`", g, got, x.pattern)
		}

	}
}

func TestParseError(t *testing.T) {
	invalid := []string{
		"",
		"a[",
	}
	for _, pattern := range invalid {
		_, err := Parse(pattern)
		if err == nil {
			t.Errorf("expected error for glob pattern %q but got nil", pattern)
		}
	}
}

type setTest struct {
	patterns []string
	basename string
	match    bool
}

var setTests = []setTest{
	{[]string{"!*_test.go", "*.go"}, "m.go", true},
	{[]string{"!*_test.go", "*.go"}, "m_test.go", true},
	{[]string{"*.go", "!*_test.go"}, "m.go", true},
	{[]string{"*.go", "!*_test.go"}, "m_test.go", false},
	{[]string{"*a*", "*b*"}, "m_test.go", false},
}

func TestGlobSetMatch(t *testing.T) {
	for _, x := range setTests {
		g, err := NewGlobSet(x.patterns...)
		if err != nil {
			t.Fatal(err)
		}
		got := g.Match(x.basename)
		if got != x.match {
			t.Errorf("%s: Match(%q) = %t; want: %t", g, x.basename, got, x.match)
		}
	}
}

func TestGlobSetReplace(t *testing.T) {
	for _, x := range setTests {
		g, err := NewGlobSet("!*")
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Replace(x.patterns); err != nil {
			t.Fatal(err)
		}
		got := g.Match(x.basename)
		if got != x.match {
			t.Errorf("%s: Match(%q) = %t; want: %t", g, x.basename, got, x.match)
		}
	}
}

func TestGlobSetGetSlice(t *testing.T) {
	g, err := NewGlobSet("*foo*.[ch]", "*.go", `*"*`)
	if err != nil {
		t.Fatal(err)
	}
	s := g.String()
	want := `["*foo*.[ch]", "*.go", "*\"*"]`
	if s != want {
		t.Errorf("%s: String() = `%s`; want: `%s`", g, s, want)
	}
}

func TestGlobSetString(t *testing.T) {
	for _, x := range setTests {
		g, err := NewGlobSet(x.patterns...)
		if err != nil {
			t.Fatal(err)
		}
		got := g.GetSlice()
		if !reflect.DeepEqual(got, x.patterns) {
			t.Errorf("GetSlice() = `%s`; want: `%s`", got, x.patterns)
		}
	}
}

func benchmarkFilepathGlob(b *testing.B, pattern, path string) {
	for i := 0; i < b.N; i++ {
		ok, _ := filepath.Match(pattern, path)
		if !ok {
			b.Fatal("bad match")
		}
	}
}

func BenchmarkFilepathGlob(b *testing.B) {
	b.Run("Prefix", func(b *testing.B) {
		benchmarkFilepathGlob(b, "package_*", "package_test.go")
	})
	b.Run("Suffix", func(b *testing.B) {
		benchmarkFilepathGlob(b, "*.go", "package_test.go")
	})
	b.Run("Contains", func(b *testing.B) {
		benchmarkFilepathGlob(b, "*_test.*", "package_test.go")
	})
	b.Run("Exact", func(b *testing.B) {
		benchmarkFilepathGlob(b, "package_test.go", "package_test.go")
	})
}

func benchmarkGlob(b *testing.B, pattern, path string) {
	g, err := Parse(pattern)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if !g.Match(path) {
			b.Fatal("bad match")
		}
	}
}

func BenchmarkGlob(b *testing.B) {
	b.Run("Prefix", func(b *testing.B) {
		benchmarkGlob(b, "package_*", "package_test.go")
	})
	b.Run("Suffix", func(b *testing.B) {
		benchmarkGlob(b, "*.go", "package_test.go")
	})
	b.Run("Contains", func(b *testing.B) {
		benchmarkGlob(b, "*_test.*", "package_test.go")
	})
	b.Run("Exact", func(b *testing.B) {
		benchmarkGlob(b, "package_test.go", "package_test.go")
	})
}

// TODO: regexp is faster in some cases
func BenchmarkRegexp(b *testing.B) {
	b.Skip("TODO: remove me")
	// re := regexp.MustCompile(`.*\.[ch]$`)
	re := regexp.MustCompile(`^some_header.h$`)
	for i := 0; i < b.N; i++ {
		// re.MatchString("header.h")
		re.MatchString("some_header.h")
	}
}
