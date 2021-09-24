package main

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompleteFiles(t *testing.T) {
	exp := []string{
		"d_01", "d_02", "d_03", "d_04", "d_05", "d_06", "d_07", "d_08",
		"f_01.txt", "f_02.txt", "f_03.txt", "f_04.txt", "f_05.txt", "f_06.txt",
		"f_07.txt", "f_08.txt", "f_09.txt", "f_10.txt", "f_11.txt", "f_12.txt",
		"f_13.txt", "f_14.txt", "f_15.txt", "f_16.txt", "f_17.txt", "f_18.txt",
		"f_19.txt", "f_20.txt", "f_21.txt", "f_22.txt", "f_23.txt", "f_24.txt",
		"f_25.txt", "f_26.txt", "f_27.txt", "f_28.txt", "f_29.txt", "f_30.txt",
		"f_31.txt", "f_32.txt", "ld_01", "lf_01.txt", "s_01.sh",
	}
	for i, s := range exp {
		exp[i] = "testdata/" + s
		if strings.HasPrefix(s, "d_") || strings.HasPrefix(s, "ld_") {
			exp[i] += "/"
		}
	}

	t.Run("Files", func(t *testing.T) {
		assert.Equal(t, exp, CompleteFiles("testdata/"))
	})

	t.Run("Dir", func(t *testing.T) {
		assert.Equal(t, []string{"testdata/"},
			CompleteFiles("testdata"))
	})
}

func TestCompleteDirs(t *testing.T) {
	exp := []string{
		"d_01", "d_02", "d_03", "d_04", "d_05", "d_06", "d_07", "d_08", "ld_01",
	}
	for i, s := range exp {
		exp[i] = "testdata/" + s + "/"
	}
	assert.Equal(t, exp, CompleteDirs("testdata/"))
}

func TestHasPrefix(t *testing.T) {
	type Test struct {
		S, Prefix string
		Equal     bool
	}
	var tests = []struct {
		S, Prefix string
	}{
		{"", ""},
		{"a", "a"},
		{"a", "A"},
		{"a1", "A1"},
		{"a", "b"},
		{"a", "B"},
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		for _, x := range tests {
			got := HasPrefix(x.S, x.Prefix)
			exp := strings.HasPrefix(strings.ToLower(x.S), strings.ToLower(x.Prefix))
			assert.Equalf(t, exp, got, "%+x", x)
		}
	} else {
		for _, x := range tests {
			got := HasPrefix(x.S, x.Prefix)
			exp := strings.HasPrefix(x.S, x.Prefix)
			assert.Equalf(t, exp, got, "%+x", x)
		}
	}
}

func BenchmarkCompleteFiles(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CompleteFiles("testdata")
	}
}
