package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// self.wordchars = ('abcdfeghijklmnopqrstuvwxyz'
//                   'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_')
// if self.posix:
//     self.wordchars += ('ßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ'
//                        'ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞ')

type Token int8

const (
	TokenNone Token = iota
	Word
)

var wordChars = [127]bool{
	'a': true, 'b': true, 'c': true, 'd': true, 'f': true, 'e': true, 'g': true,
	'h': true, 'i': true, 'j': true, 'k': true, 'l': true, 'm': true, 'n': true,
	'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true, 'u': true,
	'v': true, 'w': true, 'x': true, 'y': true, 'z': true, 'A': true, 'B': true,
	'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true, 'I': true,
	'J': true, 'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true,
	'Q': true, 'R': true, 'S': true, 'T': true, 'U': true, 'V': true, 'W': true,
	'X': true, 'Y': true, 'Z': true, '0': true, '1': true, '2': true, '3': true,
	'4': true, '5': true, '6': true, '7': true, '8': true, '9': true, '_': true,
}

func IsWordChar(r rune) bool {
	return r < int32(len(wordChars)) && wordChars[r]
}

func IsPosixWordChar(r rune) bool {
	if r < int32(len(wordChars)) {
		return wordChars[r]
	}
	switch r {
	case 'ß', 'à', 'á', 'â', 'ã', 'ä', 'å', 'æ', 'ç', 'è', 'é', 'ê', 'ë', 'ì',
		'í', 'î', 'ï', 'ð', 'ñ', 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'ù', 'ú', 'û',
		'ü', 'ý', 'þ', 'ÿ', 'À', 'Á', 'Â', 'Ã', 'Ä', 'Å', 'Æ', 'Ç', 'È', 'É',
		'Ê', 'Ë', 'Ì', 'Í', 'Î', 'Ï', 'Ð', 'Ñ', 'Ò', 'Ó', 'Ô', 'Õ', 'Ö', 'Ø',
		'Ù', 'Ú', 'Û', 'Ü', 'Ý', 'Þ':
		return true
	}
	return false
}

type deque struct {
	list list.List
}

func (d *deque) Len() int { return d.list.Len() }

func (d *deque) appendLeft(tok string) {
	d.list.PushFront(tok)
}

func (d *deque) popLeft() string {
	return d.list.Remove(d.list.Front()).(string)
}

type Shlex struct {
	r               io.RuneReader
	pushback        deque
	Posix           bool
	WhitespaceSplit bool
}

func (s *Shlex) GetToken() (string, bool) {
	if s.pushback.Len() != 0 {
		return s.pushback.popLeft(), true
	}

	return "", false
}

func main() {
	const posixWords = "ßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ" + "ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞ"
	const words = "abcdfeghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

	// r := strings.NewReader(words)
	// r.ReadRune()

	s := words
	for len(s) > 0 {
		n := 7
		if n > len(s) {
			n = len(s)
		}
		a := s[:n]
		s = s[n:]
		for _, r := range a {
			fmt.Printf("'%c': Word, ", r)
		}
		fmt.Print("\n")
	}
	// for _, r := range words {
	// 	fmt.Printf("'%c'\n", r)
	// }
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
