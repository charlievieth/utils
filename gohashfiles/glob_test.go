package main

import "testing"

type GlobTest struct {
	pattern, path string
	match         bool
}

var GlobTests = []GlobTest{
	{".git", ".git", true},
	{"!.git", ".git", false},
	{"!.git", ".hg", true},

	{"b*", "/a/bb", true},
	{"b*", "/a/c", false},
	{"/a/b*", "/a/bb", true},
	{"/a/b*", "/a/c", false},
}

func TestGlob(t *testing.T) {
	for _, test := range GlobTests {
		g, err := NewGlob(test.pattern)
		if err != nil {
			t.Fatalf("%+v: %v", test, err)
		}
		got := g.Match(test.path)
		if got != test.match {
			t.Errorf("Match(%q, %q) = %t; want: %t",
				test.pattern, test.path, got, test.match)
		}
	}
}

func TestGlobSet(t *testing.T) {
	var gs GlobSet
	gs.Set("*.go")
	gs.Set("!*_test.go")
	tests := map[string]bool{
		"main.go":      true,
		"main_test.go": false,
	}
	for path, want := range tests {
		got := gs.Match(path)
		if got != want {
			t.Errorf("Match(%q) = %t; want: %t", path, got, want)
		}
	}
}
