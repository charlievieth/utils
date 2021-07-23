package main

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"
)

type TestCase struct {
	Input  string   `json:"input"`
	Output []string `json:"output"`
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

// WARN: this only parse a SINGLE value right now
func pythonShlex(t testing.TB, posix bool, input string) []string {
	data, err := json.Marshal([]string{input}) // WARN
	if err != nil {
		t.Fatal(err)
	}

	args := []string{"testdata/compat.py"}
	if posix {
		args = append(args, "--posix")
	}
	args = append(args, string(data))

	out, err := exec.Command("python3", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("Error running command: %v\n### OUT\n%s\n###\n", err, string(out))
	}

	var m map[string][]string
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("Error: %s\n### Data\n%s\n###\n", err, string(out))
	}
	for _, v := range m {
		return v
	}
	t.Fatal("UNREACHABLE")
	return nil
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

func testSplit(t testing.TB, posix bool, x TestCase) (passed bool) {
	t.Helper()
	lex := ShlexFromString(x.Input)
	lex.Posix = posix
	lex.Debug = testing.Verbose()
	lex.WhitespaceSplit = true
	lex.Reset(strings.NewReader(x.Input))
	tokens, err := lex.Split()
	if err != nil {
		t.Errorf("%q: error: %v", x.Input, err)
		return false
	}
	if stringsEqual(tokens, x.Output) {
		return true
	}
	// re-run with debug output
	if !testing.Verbose() {
		lex := ShlexFromString(x.Input)
		lex.Posix = posix
		lex.Debug = true
		lex.WhitespaceSplit = true
		other, err := lex.Split()
		if err != nil {
			t.Errorf("%q: error: %v", x.Input, err)
			return false
		}
		if !stringsEqual(tokens, other) {
			t.Fatal("Debug changed output!")
		}
	}
	return compareOutput(t, x.Input, tokens, x.Output)
}

func TestSplitGoogleCompat(t *testing.T) {
	const googleTestString = "one two \"three four\" \"five \\\"six\\\"\" seven#eight # nine # ten\n eleven 'twelve\\' thirteen=13 fourteen/14"

	t.Run("NoPosix", func(t *testing.T) {
		want := []string{
			"one",
			"two",
			`"three four"`,
			`"five \"`,
			`six\""`,
			"seven",
			"eleven",
			"'twelve\\'",
			"thirteen=13",
			"fourteen/14",
		}
		testSplit(t, false, TestCase{Input: googleTestString, Output: want})
	})

	t.Run("Posix", func(t *testing.T) {
		want := []string{
			"one",
			"two",
			"three four",
			"five \"six\"",
			"seven",
			"eleven",
			"twelve\\",
			"thirteen=13",
			"fourteen/14",
		}
		testSplit(t, true, TestCase{Input: googleTestString, Output: want})
	})
}

func TestPythonCompat_NoPosix(t *testing.T) {
	var failed int
	tests := LoadTestCase(t, "testdata/data.json")
	for _, x := range tests {
		// WARN WARN
		// output := pythonShlex(t, false, x.Input)
		// x.Output = output

		if !testSplit(t, false, x) {
			failed++
		}
		continue

		lex := ShlexFromString(x.Input)

		// WARN: tests fail when WhitespaceSplit == true
		// this should not happen
		//
		// lex.WhitespaceSplit = true

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
	}
	if t.Failed() {
		t.Logf("### failed=%d passed=%d", failed, len(tests)-failed)
	}
}

func TestPythonCompat_Posix(t *testing.T) {
	var failed int
	tests := LoadTestCase(t, "testdata/posix_data.json")
	for _, x := range tests {
		testSplit(t, true, x)
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

// WARN: debug only
/*
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
*/

/*
type NoError struct {
	*testing.T
}

func (e NoError) Error(args ...interface{}) {
	e.Helper()
	e.Log(args...)
}

func (e NoError) Errorf(format string, args ...interface{}) {
	e.Helper()
	e.Logf(format, args...)
}

func (e NoError) Fatal(args ...interface{}) {
	e.Helper()
	e.Log(args...)
}

func (e NoError) Fatalf(format string, args ...interface{}) {
	e.Helper()
	e.Logf(format, args...)
}
*/

/*
func printDiff(t testing.TB, got, want []string) {
	f1, err := os.Create(t.TempDir() + "/got.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()
	f2, err := os.Create(t.TempDir() + "/want.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	for _, s := range got {
		fmt.Fprintln(f1, s)
	}
	for _, s := range want {
		fmt.Fprintln(f2, s)
	}
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "diff", "--color=always", "--no-index",
		f1.Name(), f2.Name())
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Error running %q: %v\n### OUT\n%s\n###\n", cmd.Args, err, string(out))
	}
	fmt.Println("Diff:")
	os.Stdout.Write(out)
}
*/
