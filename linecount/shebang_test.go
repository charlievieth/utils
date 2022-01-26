package main

import "testing"

type shebangTest struct {
	In, Exp string
	Failed  bool
}

var shebangTests = []shebangTest{
	{
		In:  "#!/bin/sh",
		Exp: "sh",
	},
	{
		In:  "#!/usr/local/bin/bash",
		Exp: "bash",
	},
	{
		In:  "#! \t /usr/local/bin/bash ",
		Exp: "bash",
	},
	{
		In:  "#!/usr/bin/env -- python",
		Exp: "python",
	},
	{
		In:  "#!/usr/bin/env -S /usr/local/bin/php -n -q -dsafe_mode=0",
		Exp: "php",
	},
	{
		In:  "#!/usr/bin/env -S PYTHONPATH=/opt/custom/modules/:$@{PYTHONPATH@} python",
		Exp: "python",
	},
	// This is non-standard, but allow it
	{
		In:  "#!/usr/bin/env python # foo this",
		Exp: "python",
	},

	// Command line options
	{
		In:  "#!/usr/bin/perl -w",
		Exp: "perl",
	},
	{
		In:  "#!/usr/bin/perl -w -s",
		Exp: "perl",
	},
	{
		In:  "#!/bin/bash -x -e  # Common shebang errors",
		Exp: "bash",
	},
	{
		In:  "#!/usr/bin/env -i \t-- python -V",
		Exp: "python",
	},
	{
		In:  "#!/bin/sh -",
		Exp: "sh",
	},
	{
		In:  "#!/bin/rc -e",
		Exp: "rc",
	},
	{
		In:  "#!/usr/bin/env -S perl -w -T",
		Exp: "perl",
	},
	{
		In:  "#!/usr/bin/env ruby --enable-frozen-string-literal --disable=gems,did_you_mean,rubyopt",
		Exp: "ruby",
	},
	{
		In:  "#!/usr/bin/env /usr/bin/ruby --enable-frozen-string-literal --disable=gems,did_you_mean,rubyopt",
		Exp: "ruby",
	},

	// Errors
	{
		In:     "#!////",
		Failed: true,
	},
	{
		In:     "#!:strength +0",
		Failed: true,
	},
	{
		In:     "    #!/usr/bin/ruby -pi.bak",
		Failed: true,
	},
}

func TestParseShebang(t *testing.T) {
	for i, x := range shebangTests {
		got, ok := ParseShebang([]byte(x.In))
		if got != x.Exp || ok != !x.Failed {
			t.Errorf("%d: ParseShebang(%q) = %q, %t want: %q, %t", i, x.In, got, ok, x.Exp, !x.Failed)
		}
	}
}

func TestExtractShebang(t *testing.T) {
	for _, x := range shebangTests {
		for _, in := range []string{x.In, x.In + "\n\necho foo!\n"} {
			got := ExtractShebang([]byte(in))
			if got != x.Exp {
				t.Errorf("ExtractShebang(%q) = %q want: %q", in, got, x.Exp)
			}
		}
	}
}

func TestTrimSpaceLeft(t *testing.T) {
	tests := []struct {
		In, Exp string
	}{
		{"\t ", ""},
		{"\t a", "a"},
		{"\t a ", "a "},
		{"a", "a"},
		{"", ""},
	}
	for _, x := range tests {
		got := string(trimSpaceLeft([]byte(x.In)))
		if got != x.Exp {
			t.Errorf("%q: got: %q want: %q", x.In, got, x.Exp)
		}
	}
}

func TestTrimSpaceRight(t *testing.T) {
	tests := []struct {
		In, Exp string
	}{
		{" \t\n", ""},
		{"a \t\n", "a"},
		{" a \t\n", " a"},
		{"a", "a"},
		{"", ""},
	}
	for _, x := range tests {
		got := string(trimSpaceRight([]byte(x.In)))
		if got != x.Exp {
			t.Errorf("%q: got: %q want: %q", x.In, got, x.Exp)
		}
	}
}

func TestBasename(t *testing.T) {
	tests := []struct {
		In, Exp string
	}{
		{"", ""},
		{"/", ""},
		{"///", ""},
		{"a", "a"},
		{"a/b", "b"},
	}
	for _, x := range tests {
		got := string(basename([]byte(x.In)))
		if got != x.Exp {
			t.Errorf("filepathBase(%q) = %q want: %q", x.In, got, x.Exp)
		}
	}
}

func BenchmarkParseShebang(b *testing.B) {
	b.Run("Short", func(b *testing.B) {
		line := []byte("#!/bin/sh")
		for i := 0; i < b.N; i++ {
			ParseShebang(line)
		}
	})

	b.Run("Env", func(b *testing.B) {
		line := []byte("#!/usr/bin/env bash")
		for i := 0; i < b.N; i++ {
			ParseShebang(line)
		}
	})

	b.Run("RubyWithOpts", func(b *testing.B) {
		line := []byte("#!/usr/bin/env ruby --enable-frozen-string-literal --disable=gems,did_you_mean,rubyopt")
		for i := 0; i < b.N; i++ {
			ParseShebang(line)
		}
	})
}
