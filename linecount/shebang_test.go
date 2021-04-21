package main

import "testing"

type ShebangTest struct {
	In, Exp string
	Err     error
}

var exeTests = []ShebangTest{
	{
		In:  "#!/usr/bin/env python # foo this",
		Exp: "python",
	},
	{
		In:  "#!/usr/bin/env -- python",
		Exp: "python",
	},
	{
		In:  "#!/usr/local/bin/bash",
		Exp: "bash",
	},
	{
		In:  "#! \t /usr/local/bin/bash ",
		Exp: "bash",
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
		In:  "    #!/usr/bin/ruby -pi.bak",
		Exp: "ruby",
	},

	// Errors
	{
		In:  "#!////",
		Err: ErrInvalidShebang,
	},
	{
		In:  "#!:strength +0",
		Err: ErrInvalidShebang,
	},
}

func TestParseShebangExe(t *testing.T) {
	for _, x := range exeTests {
		got, err := ParseShebangExe([]byte(x.In))
		if err != x.Err {
			t.Errorf("%q: Error: got: %v want: %v", x.In, err, x.Err)
			continue
		}
		if got != x.Exp {
			t.Errorf("%q: Exe: got: %q want: %q", x.In, got, x.Exp)
		}
	}
}

var extractTests = []ShebangTest{
	{
		In: `
# this is comment
# #!/usr/FAKE
#!/usr/bin/env bash

echo "Hello!"
exit 1
`,
		Exp: "bash",
	},
}

func TestExtractShebang(t *testing.T) {
	for _, x := range exeTests {
		got := ExtractShebang([]byte(x.In))
		if got != x.Exp {
			t.Errorf("%q: Exe: got: %q want: %q", x.In, got, x.Exp)
		}
	}
}

func TestTrimSpaceLeft(t *testing.T) {
	tests := map[string]string{
		"\t /usr/local/bin/bash": "/usr/local/bin/bash",
	}
	for in, exp := range tests {
		got := string(trimSpaceLeft([]byte(in)))
		if got != exp {
			t.Errorf("%q: got: %q want: %q", in, got, exp)
		}
	}
}
