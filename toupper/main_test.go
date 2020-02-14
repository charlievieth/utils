package main

import "testing"

type StringTest struct {
	in, out string
}

var upperTests = []StringTest{
	{"", ""},
	{"ONLYUPPER", "ONLYUPPER"},
	{"abc", "ABC"},
	{"AbC123", "ABC123"},
	{"azAZ09_", "AZAZ09_"},
	{"longStrinGwitHmixofsmaLLandcAps", "LONGSTRINGWITHMIXOFSMALLANDCAPS"},
	{"long\u0250string\u0250with\u0250nonascii\u2C6Fchars", "LONG\u2C6FSTRING\u2C6FWITH\u2C6FNONASCII\u2C6FCHARS"},
	{"\u0250\u0250\u0250\u0250\u0250", "\u2C6F\u2C6F\u2C6F\u2C6F\u2C6F"}, // grows one byte per char
}

var TestMap Map

// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func runBytesTests(t *testing.T, f func([]byte) []byte, funcName string, testCases []StringTest) {
	for _, tc := range testCases {
		actual := string(f([]byte(tc.in)))
		if actual != tc.out {
			t.Errorf("%s(%q) = %q; want %q", funcName, tc.in, actual, tc.out)
		}
	}
}

// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func runStringTests(t *testing.T, f func(string) []byte, funcName string, testCases []StringTest) {
	for _, tc := range testCases {
		actual := string(f(tc.in))
		if actual != tc.out {
			t.Errorf("%s(%q) = %q; want %q", funcName, tc.in, actual, tc.out)
		}
	}
}

func TestBytesToUpper(t *testing.T) {
	runBytesTests(t, TestMap.ToUpperBytes, "ToUpperBytes", upperTests)
}

func TestStringToUpper(t *testing.T) {
	runStringTests(t, TestMap.ToUpperString, "ToUpperString", upperTests)
}
