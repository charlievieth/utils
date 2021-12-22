package main

import "testing"

// FD
// "\x1b[38;2;129;162;190m./\x1b[0m\x1b[38;2;129;162;190mgolang.org\x1b[0m"
// "\x1b[38;2;129;162;190m./\x1b[0m\x1b[38;2;129;162;190mmvdan.cc\x1b[0m"
// "\x1b[38;2;129;162;190m./\x1b[0m\x1b[38;2;129;162;190mgithub.com\x1b[0m"
// "\x1b[38;2;129;162;190m./\x1b[0m\x1b[38;2;129;162;190mgo.uber.org\x1b[0m"
// "\x1b[38;2;129;162;190m./golang.org/\x1b[0m\x1b[38;2;129;162;190mx\x1b[0m"
// "\x1b[38;2;129;162;190m./mvdan.cc/\x1b[0m\x1b[38;2;129;162;190msh\x1b[0m"
// "\x1b[38;2;129;162;190m./go.uber.org/\x1b[0m\x1b[38;2;129;162;190mmultierr\x1b[0m"
// "\x1b[38;2;129;162;190m./golang.org/x/\x1b[0m\x1b[38;2;129;162;190mmod\x1b[0m"
// "\x1b[38;2;129;162;190m./mvdan.cc/sh/\x1b[0m\x1b[38;2;129;162;190mvendor\x1b[0m"
// "\x1b[38;2;129;162;190m./github.com/\x1b[0m\x1b[38;2;129;162;190mmdempsky\x1b[0m"

// RG
// "\x1b[0m\x1b[35mgosort/main.go\x1b[0m"
// "\x1b[0m\x1b[35masm-hash-test/asm_repl-1_amd64.s\x1b[0m"
// "\x1b[0m\x1b[35masm-hash-test/repl-1.go\x1b[0m"
// "\x1b[0m\x1b[35mgomatch/main.go\x1b[0m"
// "\x1b[0m\x1b[35mcmdbench/main.go\x1b[0m"
// "\x1b[0m\x1b[35mgoxor.tar.zst\x1b[0m"
// "\x1b[0m\x1b[35mfail-dupe/main.go\x1b[0m"
// "\x1b[0m\x1b[35mfail-dupe/test.bash\x1b[0m"
// "\x1b[0m\x1b[35mformat-json/main.go\x1b[0m"
// "\x1b[0m\x1b[35mformat-json/.gitignore\x1b[0m"

func TestReplaceANSII(t *testing.T) {
	tests := []struct {
		In, Exp string
	}{
		{
			In:  "\x1b[38;2;129;162;190m./\x1b[0m\x1b[38;2;129;162;190mgolang.org\x1b[0m",
			Exp: "./golang.org",
		},
		{
			In:  "\x1b[0m\x1b[35masm-hash-test/asm_repl-1_amd64.s\x1b[0m",
			Exp: "asm-hash-test/asm_repl-1_amd64.s",
		},
	}
	for _, x := range tests {
		got := string(ReplaceANSII([]byte(x.In)))
		if got != string(x.Exp) {
			t.Errorf("ReplaceANSII(%q) = %q want: %q", x.In, got, x.Exp)
		}
	}
}

func BenchmarkReplaceANSII(b *testing.B) {
	// "\x1b[38;2;129;162;190m./github.com/\x1b[0m\x1b[38;2;129;162;190mmdempsky\x1b[0m"
	// "\x1b[0m\x1b[35masm-hash-test/asm_repl-1_amd64.s\x1b[0m"
	b.Run("FD", func(b *testing.B) {
		s := []byte("\x1b[38;2;129;162;190m./github.com/\x1b[0m\x1b[38;2;129;162;190mmdempsky\x1b[0m")
		b.SetBytes(int64(len(s)))
		for i := 0; i < b.N; i++ {
			ReplaceANSII(s)
		}
	})
	b.Run("RG", func(b *testing.B) {
		s := []byte("\x1b[0m\x1b[35masm-hash-test/asm_repl-1_amd64.s\x1b[0m")
		b.SetBytes(int64(len(s)))
		for i := 0; i < b.N; i++ {
			ReplaceANSII(s)
		}
	})
	b.Run("NoColor", func(b *testing.B) {
		s := []byte("asm-hash-test/asm_repl-1_amd64.s")
		b.SetBytes(int64(len(s)))
		for i := 0; i < b.N; i++ {
			ReplaceANSII(s)
		}
	})
}
