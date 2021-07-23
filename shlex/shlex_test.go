package main

import (
	"encoding/json"
	"io/ioutil"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	Input  string   `json:"input"`
	Output []string `json:"output"`
}

const googleTestString = "one two \"three four\" \"five \\\"six\\\"\" seven#eight # nine # ten\n eleven 'twelve\\' thirteen=13 fourteen/14"

func TestSplitGoogleCompat(t *testing.T) {
	// Test from: github.com/google/shlex
	t.Run("Google", func(t *testing.T) {
		defer func() {
			if e := recover(); e != nil {
				buf := make([]byte, 32*1024)
				n := runtime.Stack(buf, false)
				t.Fatalf("panic: %v\n\n%s\n", e, buf[:n])
			}
		}()
		want := []string{"one", "two", "three four", "five \"six\"", "seven#eight",
			"eleven", "twelve\\", "thirteen=13", "fourteen/14"}
		got, err := Split(googleTestString, true)
		if err != nil {
			t.Error(err)
		}
		if len(want) != len(got) {
			t.Errorf("Split(%q) -> %v. Want: %v", googleTestString, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("Split(%q)[%v] -> %v. Want: %v", googleTestString, i, got[i], want[i])
			}
		}
	})

	// Test against how Python shlex parses this input
	t.Run("Python", func(t *testing.T) {
		// JSON:
		// [
		//     "one",
		//     "two",
		//     "three four",
		//     "five \"six\"",
		//     "seven#eight",
		//     "#",
		//     "nine",
		//     "#",
		//     "ten",
		//     "eleven",
		//     "twelve\\",
		//     "thirteen=13",
		//     "fourteen/14"
		// ]
		want := []string{`one`, `two`, `three four`, `five "six"`, `seven#eight`, `#`, `nine`, `#`,
			`ten`, `eleven`, `twelve\`, `thirteen=13`, `fourteen/14`}
		got, err := Split(googleTestString, true)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, want, got)
		return
		if len(want) != len(got) {
			t.Errorf("Split(%q) -> %v. Want: %v", googleTestString, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("Split(%q)[%v] -> %v. Want: %v", googleTestString, i, got[i], want[i])
			}
		}
	})
}

func LoadTestCase(t *testing.T, name string) []TestCase {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var tests []TestCase
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatal(err)
	}
	return tests
}

func TestPythonCompat_NoPosix(t *testing.T) {
	var failed int
	tests := LoadTestCase(t, "testdata/data.json")
	for _, x := range tests {
		if x.Input == ":-) ;-)" {
			t.Logf("FIXME: %q", ":-) ;-)")
			continue
		}
		lex := ShlexFromString(x.Input)
		lex.Debug = testing.Verbose()
		got, err := lex.Split()
		if err != nil {
			t.Errorf("%q: error: %v", x.Input, err)
			failed++
			continue
		}
		if !compareOutput(t, x.Input, got, x.Output) {
			failed++
		}
		// if !reflect.DeepEqual(got, x.Output) {
		// 	t.Errorf("%q: got: %q want: %q", x.Input, got, x.Output)
		// 	failed++
		// }
	}
	if t.Failed() {
		t.Logf("### failed=%d passed=%d", failed, len(tests)-failed)
	}
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareOutput(t testing.TB, input string, got, want []string) bool {
	t.Helper()
	if !stringsEqual(got, want) {
		t.Errorf("%q:\n\tgot:  %q\n\twant: %q", input, got, want)
		return false
	}
	return true
}

// WARN: debug only
func TestPythonCompat_One(t *testing.T) {
	// POSIX
	// "foo '' '' '' bar": got: ["foo" "bar"] want: ["foo" "" "" "" "bar"]
	const input = `foo '' '' '' bar`
	// exp := []string{""}
	exp := []string{"foo", "", "", "", "bar"}
	lex := ShlexFromString(input)
	lex.Posix = true
	lex.Debug = true
	got, err := lex.Split()
	if err != nil {
		t.Fatal(err)
	}
	compareOutput(t, input, got, exp)
	// if !reflect.DeepEqual(got, exp) {
	// 	t.Errorf("%q: got: %q want: %q", input, got, exp)
	// }
}

func TestPythonCompat_Posix(t *testing.T) {
	var failed int
	tests := LoadTestCase(t, "testdata/posix_data.json")
	for _, x := range tests {
		lex := ShlexFromString(x.Input)
		lex.Posix = true
		lex.Debug = testing.Verbose()
		got, err := lex.Split()
		if err != nil {
			t.Errorf("%q: error: %v", x.Input, err)
			failed++
			continue
		}
		if !compareOutput(t, x.Input, got, x.Output) {
			failed++
		}
		// if !reflect.DeepEqual(got, x.Output) {
		// 	t.Errorf("%q: got: %q want: %q", x.Input, got, x.Output)
		// 	failed++
		// }
	}
	if t.Failed() {
		t.Logf("### failed=%d passed=%d", failed, len(tests)-failed)
	}
}

func BenchmarkSplit(b *testing.B) {
	const testString = "one two \"three four\" \"five \\\"six\\\"\" seven#eight # nine # ten\n eleven 'twelve\\' thirteen=13 fourteen/14"
	for i := 0; i < b.N; i++ {
		Split(testString, false)
	}
}
