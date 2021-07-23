package main

import (
	"bytes"
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charlievieth/utils/shlex/token"
)

// self.wordchars = ('abcdfeghijklmnopqrstuvwxyz'
//                   'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_')
// if self.posix:
//     self.wordchars += ('ßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ'
//                        'ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝÞ')

var debug = log.New(os.Stderr, "shlex: ", log.Lshortfile)

type deque struct {
	list list.List
}

func (d *deque) Len() int { return d.list.Len() }

func (d *deque) append(tok rune) {
	d.list.PushBack(tok)
}

func (d *deque) appendLeft(tok rune) {
	d.list.PushFront(tok)
}

func (d *deque) pop() rune {
	return d.list.Remove(d.list.Back()).(rune)
}

func (d *deque) popLeft() rune {
	return d.list.Remove(d.list.Front()).(rune)
}

// TODO: use Options to configure Shlex
// type Option interface {
// 	apply(s *Shlex)
// }

type Shlex struct {
	r             io.RuneReader
	pushback      deque
	pushbackChars deque

	// state token.Token // WARN: is this correct?
	state rune

	// token rune
	token bytes.Buffer

	lineno           int
	Debug            bool
	Posix            bool // TODO: default to true
	WhitespaceSplit  bool // TODO: default to true
	PunctuationChars bool
	Commenters       bool
}

func (s *Shlex) Reset(r io.RuneReader) {
	s.r = r
	s.pushback = deque{}      // TODO: reset
	s.pushbackChars = deque{} // TODO: reset
	s.state = ' '
	s.token.Reset()
	s.lineno = 0
}

func (s *Shlex) debugf(format string, a ...interface{}) {
	if s.Debug {
		debug.Output(2, fmt.Sprintf(format, a...))
	}
}

// TODO: use our own EOF ???
// var EOF = io.EOF

// GetToken returns io.EOF when there are no more tokens
func (s *Shlex) GetToken() (TokenLiteral, error) {
	if s.pushback.Len() != 0 {
		tok := string(s.pushback.popLeft())
		s.debugf("popping token %s", tok)
		return TokenLiteral{Token: tok, Valid: true}, nil
	}
	raw, err := s.ReadToken()
	if err != io.EOF {
		s.debugf("token=%s", raw.Token)
	} else {
		s.debugf("token=EOF")
	}
	return raw, err
}

func (s *Shlex) readline() (err error) {
	for {
		r, _, e := s.r.ReadRune()
		if r == '\n' || e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	return err // WARN
}

func (s *Shlex) isWord(r rune) bool {
	if s.Posix {
		return token.IsPosixWord(r)
	}
	return token.IsWord(r)
}

func (s *Shlex) isComment(r rune) bool {
	// return s.Commenters && token.IsComment(r)
	return token.IsComment(r)
}

func (s *Shlex) isPunctuation(r rune) bool {
	return s.PunctuationChars && token.IsPunctuation(r)
}

func (s *Shlex) classify(r rune) token.Token {
	c := token.Classify(r)
	if !s.PunctuationChars && c == token.Punctuation {
		c = token.None
	} else if !s.Commenters && c == token.Comment {
		// c = token.None
	}
	// if c == token.Punctuation && !s.PunctuationChars {
	// 	c = token.None
	// }
	return c
}

type TokenLiteral struct {
	Token string
	Valid bool
}

func (s *Shlex) ReadToken() (TokenLiteral, error) {
	// s.token.Reset() // WARN WARN WARN

	quoted := false
	var escapedState rune
	_ = escapedState
	var err error
Loop:
	for {
		var nextchar rune
		if s.PunctuationChars && s.pushbackChars.Len() > 0 {
			nextchar = s.pushbackChars.pop()
		} else {
			nextchar, _, err = s.r.ReadRune()
			if err != nil {
				if err == io.EOF {
					// s.debugf("EOF")
				} else {
					s.debugf("error: %v\n", err)
				}
				// WARN: we should not need this
				// break Loop
			}
		}
		if nextchar == '\n' {
			s.lineno++
		}
		if s.Debug {
			s.debugf("in state '%c' I see character: '%c'", s.state, nextchar)
		}
		switch s.classify(s.state) {
		case token.None:
			s.token.Reset() // WARN
			break Loop
		case token.Whitespace:
			// TODO: need to pass Posix/Non-Posix
			if nextchar == 0 {
				s.state = 0
				break Loop
			}
			class := s.classify(nextchar)
			switch class {
			// case token.None:
			// s.debugf("token.None") // WARN
			case token.Whitespace:
				s.debugf("shlex: I see whitespace in whitespace state") // WARN
				if s.token.Len() > 0 || (s.Posix && quoted) {
					break Loop // emit current token
				} else {
					continue Loop
				}
			case token.Comment:
				s.debugf("READLINE")
				_ = s.readline() // WARN: handle error
				s.lineno++
			case token.Word:
				// s.token = nextchar
				s.token.WriteRune(nextchar)
				s.state = 'a' // WARN
			case token.Quote:
				if !s.Posix {
					s.token.WriteRune(nextchar)
				}
				s.state = nextchar
			default:
				switch {
				case s.Posix && class == token.Escape:
					escapedState = 'a' // WARN
					s.state = nextchar
				case s.WhitespaceSplit:
					// s.token = nextchar
					s.token.WriteRune(nextchar)
					s.state = 'a' // WARN
				case s.PunctuationChars && class == token.Punctuation:
					// s.token = nextchar
					s.token.WriteRune(nextchar)
					s.state = 'c' // WARN
				default:
					// s.token = nextchar
					s.token.WriteRune(nextchar)
					if s.token.Len() > 0 || (s.Posix && quoted) {
						break Loop // emit current token
					} else {
						continue Loop
					}
				}
			}
		case token.Quote:
			// fmt.Println("HIT:", string(nextchar)) // WARN
			quoted = true
			if nextchar == 0 {
				s.debugf("I see EOF in quotes state")
				return TokenLiteral{}, errors.New("no closing quotation")
			}
			switch {
			case nextchar == s.state:
				// fmt.Printf("HIT 2: '%c'\n", nextchar) // WARN
				if !s.Posix {
					s.token.WriteRune(nextchar)
					s.state = ' '
					break Loop // WARN
				} else {
					// WARN WARN WARN WARN
					//
					// We might not be re-entering the loop here so
					// might be dropping the last token
					//
					// WARN WARN WARN WARN
					s.state = 'a'
				}
			case s.Posix && token.IsEscape(nextchar) && token.IsEscapedQuote(s.state):
				escapedState = s.state
				s.state = nextchar
			default:
				s.token.WriteRune(nextchar)
			}
		case token.Escape:
			if nextchar == 0 {
				s.debugf("I see EOF in escaped state")
				return TokenLiteral{}, errors.New("no escaped character")
			}
			// In posix shells, only the quote itself or the escape
			// character may be escaped within quotes.
			if token.IsQuote(escapedState) && nextchar != s.state && nextchar != escapedState {
				s.token.WriteRune(s.state)
			}
			s.token.WriteRune(nextchar)
			s.state = escapedState

			// WARN: this is handled outside of the switch
			// case 'a', 'c':
		default:
			if s.state == 'a' || s.state == 'c' {
				// fmt.Printf("HIT 3: '%c' - %q\n", nextchar, s.token.String()) // WARN
				switch {
				case nextchar == 0:
					s.state = 0
					break Loop
				case token.IsWhitespace(nextchar):
					s.debugf("I see whitespace in word state")
					s.state = ' '
					if s.token.Len() > 0 || (s.Posix && quoted) {
						break Loop // emit current token
					} else {
						continue Loop
					}
				case s.isComment(nextchar):
					s.debugf("READLINE")
					s.readline() // WARN: check error
					s.lineno++
					if s.Posix {
						s.state = ' '
						if s.token.Len() > 0 || (s.Posix && quoted) {
							break Loop // emit current token
						} else {
							continue Loop
						}
					}
				case s.state == 'c':
					if s.isPunctuation(nextchar) {
						s.token.WriteRune(nextchar)
					} else {
						if token.IsWhitespace(nextchar) {
							s.pushbackChars.append(nextchar)
						}
						s.state = ' '
						break Loop
					}
				case s.Posix && token.IsQuote(nextchar):
					// fmt.Println("XXXX")
					s.state = nextchar
				case s.Posix && token.IsEscape(nextchar):
					escapedState = 'a'
					s.state = nextchar
				case s.isWord(nextchar) || token.IsQuote(nextchar) ||
					(s.WhitespaceSplit && !s.isPunctuation(nextchar)):
					s.token.WriteRune(nextchar)
				default:
					if s.PunctuationChars {
						s.pushbackChars.append(nextchar)
					} else {
						s.pushback.appendLeft(nextchar)
					}
					s.debugf("I see punctuation in word state")
					s.state = ' '
					if s.token.Len() > 0 || (s.Posix && quoted) {
						break Loop // emit current token
					} else {
						continue Loop
					}
				}
			} else {
				s.debugf("WARN: in state %c I see character: %c", s.state, nextchar)
			}
		}
	}
	if err != nil {
		s.debugf("ERROR: err=%v", err)
	}
	var result TokenLiteral
	if !(s.Posix && !quoted && s.token.Len() == 0) {
		result.Token = s.token.String()
		result.Valid = true
	}
	s.token.Reset()
	if result.Token != "" {
		s.debugf("raw token=%s", result.Token)
	} else {
		s.debugf("raw token=EOF")
	}
	// TODO: return io.EOF || something
	return result, err
}

func ShlexFromString(s string) *Shlex {
	return &Shlex{
		r:     strings.NewReader(s),
		state: ' ',
	}
}

func (s *Shlex) Split() ([]string, error) {
	var toks []string
	var err error
	for {
		var t TokenLiteral
		t, err = s.GetToken()
		// WARN WARN WARN WARN WARN WARN WARN
		//
		// Need to distinguish between "" and None
		//
		// WARN WARN WARN WARN WARN WARN WARN
		// if err == nil || err == io.EOF {
		// if t != "" || (err == nil || err == io.EOF) {
		if t.Valid {
			toks = append(toks, t.Token)
		}
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		err = nil
	}
	if i := len(toks); !s.Posix && i != 0 && toks[i-1] == "" {
		toks = toks[:i-1]
	}
	return toks, err
}

func Split(s string, posix bool) ([]string, error) {
	sh := ShlexFromString(s)
	sh.WhitespaceSplit = true
	return sh.Split()
}

func main() {
	// {
	// 	r := strings.NewReader("echo")
	// 	for {
	// 		c, _, e := r.ReadRune()
	// 		fmt.Printf("%c: %v\n", c, e)
	// 		if e != nil {
	// 			break
	// 		}
	// 	}
	// 	return
	// }

	fmt.Println(Split(`echo none 'single' "double"`, true))
	return

	lex := ShlexFromString(`echo none 'single' "double"`)
	for i := 0; i < 10; i++ {
		tok, err := lex.GetToken()
		if err != nil {
			break
		}
		fmt.Printf("%q\n", tok.Token)
	}
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

/*
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
*/
