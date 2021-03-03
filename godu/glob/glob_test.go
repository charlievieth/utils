package main

import "testing"

func TestNamedClassString(t *testing.T) {
	tests := []struct {
		Class NamedClass
		Exp   string
		Name  string
	}{
		{-1, "NamedClass(-1)", "NamedClass(-1)"},
		{ClassNone, "None", "None"},
		{ClassAlnum, "Alnum", "[:alnum:]"},
		{ClassAlpha, "Alpha", "[:alpha:]"},
		{ClassASCII, "ASCII", "[:ascii:]"},
		{ClassBlank, "Blank", "[:blank:]"},
		{ClassCntrl, "Cntrl", "[:cntrl:]"},
		{ClassDigit, "Digit", "[:digit:]"},
		{ClassGraph, "Graph", "[:graph:]"},
		{ClassLower, "Lower", "[:lower:]"},
		{ClassPrint, "Print", "[:print:]"},
		{ClassPunct, "Punct", "[:punct:]"},
		{ClassSpace, "Space", "[:space:]"},
		{ClassUpper, "Upper", "[:upper:]"},
		{ClassWord, "Word", "[:word:]"},
		{ClassXDigit, "XDigit", "[:xdigit:]"},
		{ClassXDigit + 1, "NamedClass(15)", "NamedClass(15)"},
	}
	for i, x := range tests {
		if got := x.Class.String(); got != x.Exp {
			t.Errorf("%d: String: got: %q want: %q", i, got, x.Exp)
		}
		if got := x.Class.Name(); got != x.Name {
			t.Errorf("%d: Name: got: %q want: %q", i, got, x.Name)
		}
	}
}

func TestIsNamedClass(t *testing.T) {
	tests := []string{
		"[:alnum:]",
		"[:alpha:]",
		"[:ascii:]",
		"[:blank:]",
		"[:cntrl:]",
		"[:digit:]",
		"[:graph:]",
		"[:lower:]",
		"[:print:]",
		"[:punct:]",
		"[:space:]",
		"[:upper:]",
		"[:word:]",
		"[:xdigit:]",
	}
	for _, s := range tests {
		exp := len(s) - 1
		got := isNamedClass(s)
		if got != exp {
			t.Errorf("%q: got: %d want: %d", s, got, exp)
		}
	}
}
