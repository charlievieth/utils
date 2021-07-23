package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// From: cpython/Lib/shlex.py
const WordChars = "abcdfeghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

const PosixWordChars = "ßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ" + "ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞ"

func TestIsWord(t *testing.T) {
	for _, r := range WordChars {
		if !IsWord(r) {
			t.Errorf("%c: got: %t want: %t\n", r, false, true)
		}
	}

	for _, r := range PosixWordChars {
		if IsWord(r) {
			t.Errorf("IsPosixWord: %c: got: %t want: %t\n", r, true, false)
		}
	}
}

func TestIsPosixWord(t *testing.T) {
	for _, r := range WordChars + PosixWordChars {
		if !IsPosixWord(r) {
			t.Errorf("%c: got: %t want: %t\n", r, false, true)
		}
	}
}

func TestClassify(t *testing.T) {
	t.Run("Comment", func(t *testing.T) {
		for _, r := range "#" {
			assert.Equalf(t, Comment, Classify(r), "%c", r)
		}
	})
	t.Run("Quote", func(t *testing.T) {
		for _, r := range `'"` {
			assert.Equalf(t, Quote, Classify(r), "%c", r)
		}
	})
	t.Run("Escape", func(t *testing.T) {
		for _, r := range "\\" {
			assert.Equalf(t, Escape, Classify(r), "%c", r)
		}
	})
	t.Run("Punctuation", func(t *testing.T) {
		for _, r := range `();<>|&` {
			assert.Equalf(t, Punctuation, Classify(r), "%c", r)
		}
	})
	t.Run("Whitespace", func(t *testing.T) {
		for _, r := range " \t\r\n" {
			assert.Equalf(t, Whitespace, Classify(r), "%q", string(r))
		}
	})
}
