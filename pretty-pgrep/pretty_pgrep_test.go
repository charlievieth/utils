package main

import "testing"

type ParseCommandNameTest struct {
	in, out, rest string
}

var parseCommandNameTests = []ParseCommandNameTest{
	{"ls", "ls", ""},
	{"ls -la", "ls", "-la"},
	{"testdata/Test App/Example.exe", "testdata/Test App/Example.exe", ""},
	{"testdata/Test App/Example.exe -x -a --foo", "testdata/Test App/Example.exe", "-x -a --foo"},
	{"./testdata/Test App/Example.exe", "./testdata/Test App/Example.exe", ""},
	{"testdata/Evil   App -xyz", "testdata/Evil   App", "-xyz"},
}

func TestParseCommandName(t *testing.T) {
	for i, test := range parseCommandNameTests {
		got, rest, err := ParseCommandName(test.in)
		if err != nil {
			t.Fatalf("%d: ParseCommandName(%q) = %q, %q, %v", i, test.in, got, rest, err)
		}
		if got != test.out || rest != test.rest {
			t.Errorf("%d: ParseCommandName(%q) = %q, %q; want: %q, %q",
				i, test.in, got, rest, test.out, test.rest)
		}
	}
}
