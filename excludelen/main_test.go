package main

import (
	"bytes"
	"encoding/base64"
	"testing"
)

const NoColorLogLine = "eyJ0aW1lc3RhbXAiOiIxNTIwOTgzODAyLjQyOTA0NjE1NCIsInNvdXJjZSI6InJl" +
	"cCIsIm1lc3NhZ2UiOiJyZXAuY29udGFpbmVyLW1ldHJpY3MtcmVwb3J0ZXIudGljay5nZXQtYWxsLW1ldHJ" +
	"pY3MuY29udGFpbmVyc3RvcmUtbWV0cmljcy5zdGFydGluZyIsImxvZ19sZXZlbCI6MSwiZGF0YSI6eyJzZX" +
	"NzaW9uIjoiOS41NzI0LjEuMSJ9fQo="

const ColorLogLine = "eyJ0aW1lc3RhbXAiOiIxNTIwOTgzODAyLjQyOTA0NjE1NCIsInNvdXJjZSI6IhtbMD" +
	"E7MzFtG1tLcmVwG1ttG1tLIiwibWVzc2FnZSI6IhtbMDE7MzFtG1tLcmVwG1ttG1tLLmNvbnRhaW5lci1tZ" +
	"XRyaWNzLRtbMDE7MzFtG1tLcmVwG1ttG1tLb3J0ZXIudGljay5nZXQtYWxsLW1ldHJpY3MuY29udGFpbmVy" +
	"c3RvcmUtbWV0cmljcy5zdGFydGluZyIsImxvZ19sZXZlbCI6MSwiZGF0YSI6eyJzZXNzaW9uIjoiOS41NzI" +
	"0LjEuMSJ9fQo="

func TestStripColor(t *testing.T) {
	// base64 encoded log lines (saves the trouble of dealing with JSON and
	// ANSI escape sequences)

	expected, err := base64.StdEncoding.DecodeString(NoColorLogLine)
	if err != nil {
		t.Fatal(err)
	}
	color, err := base64.StdEncoding.DecodeString(ColorLogLine)
	if err != nil {
		t.Fatal(err)
	}
	r := Reader{buf: color}
	oldBuf := make([]byte, len(r.buf))
	copy(oldBuf, r.buf)
	out := r.stripANSI(nil)
	if !bytes.Equal(expected, out) {
		t.Errorf("StripColor:\nGot: %s\nWant: %s\n", string(out), string(expected))
	}
	if !bytes.Equal(oldBuf, r.buf) {
		t.Error("StripColor: modified the underlying buffer")
	}
}

func TestPrintLen(t *testing.T) {
	// base64 encoded log lines (saves the trouble of dealing with JSON and
	// ANSI escape sequences)

	expected, err := base64.StdEncoding.DecodeString(NoColorLogLine)
	if err != nil {
		t.Fatal(err)
	}
	color, err := base64.StdEncoding.DecodeString(ColorLogLine)
	if err != nil {
		t.Fatal(err)
	}
	r := Reader{buf: color}
	n := r.PrintLen()
	if n != len(expected) {
		t.Errorf("PrintLen: Got: %d Want: %d", n, len(expected))
	}
}
