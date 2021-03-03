package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

var (
	ErrBadPattern = errors.New("syntax error in pattern")
	ErrInternal   = errors.New("internal error")
)

type NamedClass int

// ASCII character classes:
const (
	ClassNone   NamedClass = iota // None
	ClassAlnum                    //   [[:alnum:]]    alphanumeric (== [0-9A-Za-z])
	ClassAlpha                    //   [[:alpha:]]    alphabetic (== [A-Za-z])
	ClassASCII                    //   [[:ascii:]]    ASCII (== [\x00-\x7F])
	ClassBlank                    //   [[:blank:]]    blank (== [\t ])
	ClassCntrl                    //   [[:cntrl:]]    control (== [\x00-\x1F\x7F])
	ClassDigit                    //   [[:digit:]]    digits (== [0-9])
	ClassGraph                    //   [[:graph:]]    graphical (== [!-~] == [A-Za-z0-9!"#$%&'()*+,\-./:;<=>?@[\\\]^_`{|}~])
	ClassLower                    //   [[:lower:]]    lower case (== [a-z])
	ClassPrint                    //   [[:print:]]    printable (== [ -~] == [ [:graph:]])
	ClassPunct                    //   [[:punct:]]    punctuation (== [!-/:-@[-`{-~])
	ClassSpace                    //   [[:space:]]    whitespace (== [\t\n\v\f\r ])
	ClassUpper                    //   [[:upper:]]    upper case (== [A-Z])
	ClassWord                     //   [[:word:]]     word characters (== [0-9A-Za-z_])
	ClassXDigit                   //   [[:xdigit:]]   hex digit (== [0-9A-Fa-f])
)

func (n NamedClass) setASCII(r *asciiRange) {
	switch n {
	case ClassAlnum:
		// [0-9A-Za-z]
		for i := '0'; i <= '9'; i++ {
			r.chars[i] = true
		}
		for i := 'A'; i <= 'Z'; i++ {
			r.chars[i] = true
		}
		for i := 'a'; i <= 'z'; i++ {
			r.chars[i] = true
		}
	case ClassAlpha:
		// [A-Za-z]
		for i := 'A'; i <= 'Z'; i++ {
			r.chars[i] = true
		}
		for i := 'a'; i <= 'z'; i++ {
			r.chars[i] = true
		}
	case ClassASCII:
		for i := '\x00'; i <= '\x7F'; i++ {
			r.chars[i] = true
		}
	case ClassBlank:
		r.chars['\t'] = true
		r.chars[' '] = true
	case ClassCntrl:
		for i := '\x00'; i <= '\x1F'; i++ {
			r.chars[i] = true
		}
		r.chars['\x7F'] = true
	case ClassDigit:
		for i := '0'; i <= '9'; i++ {
			r.chars[i] = true
		}
	case ClassGraph:
		for i := '!'; i <= '~'; i++ {
			r.chars[i] = true
		}
	case ClassLower:
		for i := 'a'; i <= 'z'; i++ {
			r.chars[i] = true
		}
	case ClassPrint:
		r.chars[' '] = true
		r.chars['-'] = true
		r.chars['~'] = true
		// [:graph:]
		for i := '!'; i <= '~'; i++ {
			r.chars[i] = true
		}
	case ClassPunct:
		// [!-/:-@[-`{-~]
		for _, c := range []byte{'!', '-', '/', ':', '-', '@', '[', '-', '`', '{', '-', '~'} {
			r.chars[c] = true
		}
	case ClassSpace:
		r.chars['\t'] = true
		r.chars['\n'] = true
		r.chars['\v'] = true
		r.chars['\f'] = true
		r.chars['\r'] = true
		r.chars[' '] = true
	case ClassUpper:
		for i := 'A'; i <= 'Z'; i++ {
			r.chars[i] = true
		}
	case ClassWord:
		// [0-9A-Za-z_]
		for i := '0'; i <= '9'; i++ {
			r.chars[i] = true
		}
		for i := 'A'; i <= 'Z'; i++ {
			r.chars[i] = true
		}
		for i := 'a'; i <= 'z'; i++ {
			r.chars[i] = true
		}
		r.chars['_'] = true
	case ClassXDigit:
		// [0-9A-Fa-f]
		for i := '0'; i <= '9'; i++ {
			r.chars[i] = true
		}
		for i := 'A'; i <= 'F'; i++ {
			r.chars[i] = true
		}
		for i := 'a'; i <= 'f'; i++ {
			r.chars[i] = true
		}
	default:
		panic("invalid NamedClass: " + n.String())
	}
}

var namedClassStrs = [...]struct {
	str, name string
}{
	{"None", "None"},
	{"Alnum", "[:alnum:]"},
	{"Alpha", "[:alpha:]"},
	{"ASCII", "[:ascii:]"},
	{"Blank", "[:blank:]"},
	{"Cntrl", "[:cntrl:]"},
	{"Digit", "[:digit:]"},
	{"Graph", "[:graph:]"},
	{"Lower", "[:lower:]"},
	{"Print", "[:print:]"},
	{"Punct", "[:punct:]"},
	{"Space", "[:space:]"},
	{"Upper", "[:upper:]"},
	{"Word", "[:word:]"},
	{"XDigit", "[:xdigit:]"},
}

func (n NamedClass) String() string {
	if uint(n) < uint(len(namedClassStrs)) {
		return namedClassStrs[n].str
	}
	return fmt.Sprintf("NamedClass(%d)", int(n))
}

func (n NamedClass) Name() string {
	if uint(n) < uint(len(namedClassStrs)) {
		return namedClassStrs[n].name
	}
	return fmt.Sprintf("NamedClass(%d)", int(n))
}

func parseNamedClass(s string) NamedClass {
	i := strings.IndexByte(s, ']')
	if i != -1 {
		switch s[:i+1] {
		case "[:alnum:]":
			return ClassAlnum
		case "[:alpha:]":
			return ClassAlpha
		case "[:ascii:]":
			return ClassASCII
		case "[:blank:]":
			return ClassBlank
		case "[:cntrl:]":
			return ClassCntrl
		case "[:digit:]":
			return ClassDigit
		case "[:graph:]":
			return ClassGraph
		case "[:lower:]":
			return ClassLower
		case "[:print:]":
			return ClassPrint
		case "[:punct:]":
			return ClassPunct
		case "[:space:]":
			return ClassSpace
		case "[:upper:]":
			return ClassUpper
		case "[:word:]":
			return ClassWord
		case "[:xdigit:]":
			return ClassXDigit
		}
	}
	return ClassNone
}

func isNamedClass(s string) int {
	if i := strings.IndexByte(s, ']'); i != -1 {
		switch s[:i+1] {
		case "[:alnum:]", "[:alpha:]", "[:ascii:]", "[:blank:]",
			"[:cntrl:]", "[:digit:]", "[:graph:]", "[:lower:]",
			"[:print:]", "[:punct:]", "[:space:]", "[:upper:]":
			return len("[:alnum:]") - 1
		case "[:word:]":
			return len("[:word:]") - 1
		case "[:xdigit:]":
			return len("[:xdigit:]") - 1
		}
	}
	return -1
}

type Token int

const (
	Literal    Token = iota
	Any              // '?'
	ZeroOrMore       // '*'
	Range            // '['
)

var tokenStrs = [...]string{
	"Literal",
	"Any",
	"ZeroOrMore",
	"Range",
}

func (t Token) String() string {
	if uint(t) < uint(len(tokenStrs)) {
		return tokenStrs[t]
	}
	return fmt.Sprintf("Token(%d)", int(t))
}

type chunk struct {
	tok Token
	lit string
}

type chunks []chunk

func (cs chunks) match(toks ...Token) bool {
	if len(cs) != len(toks) {
		return false
	}
	for i, c := range cs {
		if c.tok != toks[i] {
			return false
		}
	}
	return true
}

func (cs chunks) literal() string {
	if len(cs) == 1 && cs[0].tok == Literal {
		return cs[0].lit
	}
	return ""
}

func (cs chunks) prefix() string {
	if cs.match(Literal, ZeroOrMore) {
		return cs[0].lit
	}
	return ""
}

func (cs chunks) extension() string {
	if cs.match(ZeroOrMore, Literal) {
		return cs[1].lit
	}
	return ""
}

func (c chunk) String() string {
	return fmt.Sprintf("{Typ:%q Lit:%q}", c.tok.String(), c.lit)
}

type asciiRange struct {
	chars   [utf8.RuneSelf]bool
	negated bool
}

func (r *asciiRange) Match(s string) int {
	if len(s) > 0 && r.chars[s[0]] {
		return 1
	}
	return -1
}

func (r *asciiRange) String() string {
	n := 0
	for _, ok := range r.chars {
		if ok {
			n++
		}
	}
	runes := make([]rune, 0, n)
	for c, ok := range r.chars {
		if ok {
			runes = append(runes, rune(c))
		}
	}
	return fmt.Sprintf("{Chars:%q Negated:%t}", runes, r.negated)
}

func (r *asciiRange) setRange(lo, hi rune) {
	if lo > hi {
		panic(fmt.Sprintf("glob: lo (%d) > hi (%d)", lo, hi))
	}
	if hi >= utf8.RuneSelf {
		panic(fmt.Sprintf("glob: non-ASCII rune: %q", r))
	}
	for i := lo; i <= hi; i++ {
		r.chars[i] = true
	}
}

func parseRange(chunk string) (*asciiRange, error) {
	// Remove only the first '['
	chunk = strings.TrimPrefix(chunk, "[")

	var r asciiRange
	if strings.HasPrefix(chunk, "^") {
		r.negated = true
		chunk = chunk[1:]
	}
	if strings.HasPrefix(chunk, "]") {
		r.chars[']'] = true
		chunk = chunk[1:]
	}

	for len(chunk) > 0 {
		switch chunk[0] {
		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				return nil, ErrBadPattern
			}
		case '[':
			if class := parseNamedClass(chunk); class != ClassNone {
				class.setASCII(&r)
				chunk = chunk[len(class.Name()):]
				continue
			}
			fallthrough
		default:
			if len(chunk) > 0 && chunk[0] == ']' {
				chunk = chunk[1:]
				break
			}
			var lo rune
			var err error
			if lo, chunk, err = getEsc(chunk); err != nil {
				return nil, err
			}
			if chunk[0] == '-' {
				var hi rune
				if hi, chunk, err = getEsc(chunk[1:]); err != nil {
					return nil, err
				}
				r.setRange(lo, hi)
			} else {
				r.chars[lo] = true
			}
		}
	}
	return &r, nil
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func getEsc(chunk string) (r rune, nchunk string, err error) {
	if len(chunk) == 0 || chunk[0] == '-' || chunk[0] == ']' {
		err = ErrBadPattern
		return
	}
	if chunk[0] == '\\' && runtime.GOOS != "windows" {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			err = ErrBadPattern
			return
		}
	}
	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = ErrBadPattern
	}
	nchunk = chunk[n:]
	if len(nchunk) == 0 {
		err = ErrBadPattern
	}
	return
}

func parse(pattern string) (chunks, error) {
	if len(pattern) == 0 {
		return nil, errors.New("glob: empty pattern")
	}
	chunks := make(chunks, 0, 2)
	var i, j int
	for ; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if j < i {
				chunks = append(chunks, chunk{
					tok: Literal,
					lit: pattern[j:i],
				})
			}
			chunks = append(chunks, chunk{
				tok: ZeroOrMore,
				lit: "*",
			})
			j = i + 1
		case '?':
			if j < i {
				chunks = append(chunks, chunk{
					tok: Literal,
					lit: pattern[j:i],
				})
			}
			chunks = append(chunks, chunk{
				tok: Any,
				lit: "?",
			})
			j = i + 1
		case '[':
			if j < i {
				chunks = append(chunks, chunk{
					tok: Literal,
					lit: pattern[j:i],
				})
			}
			o := i
			i++
			if i == len(pattern) {
				return nil, ErrBadPattern
			}
			if pattern[i] == ']' {
				i++
			}
		RangeLoop:
			for ; i < len(pattern); i++ {
				c := pattern[i]
				switch c {
				case '\\':
					i++
					if i == len(pattern) {
						return nil, ErrBadPattern
					}
				case ']':
					break RangeLoop
				case '[':
					if n := isNamedClass(pattern[i:]); n != -1 {
						i += n
					}
				}
			}
			if i == len(pattern) || pattern[i] != ']' {
				return nil, ErrBadPattern
			}
			chunks = append(chunks, chunk{
				tok: Range,
				lit: pattern[o : i+1],
			})
			j = i + 1
		case ']':
			return nil, ErrBadPattern
		case '\\':
			i++
			if i == len(pattern) {
				return nil, ErrBadPattern
			}
		}
	}
	if j < i {
		chunks = append(chunks, chunk{
			tok: Literal,
			lit: pattern[j:],
		})
	}
	return chunks, nil
}

type Glob struct {
	pattern   string
	negate    bool
	hashSlash bool // must use filepath.Match

	literal         string
	basenameLiteral string
	extension       string
	prefix          string
}

func (g *Glob) Literal() string {
	return ""
}
func (g *Glob) BasenameLiteral() string {
	return ""
}
func (g *Glob) Extension() string {
	return ""
}
func (g *Glob) Prefix() string {
	return ""
}

func matchChunks(chunks []chunk, toks ...Token) bool {
	if len(chunks) != len(toks) {
		return false
	}
	for i, c := range chunks {
		if c.tok != toks[i] {
			return false
		}
	}
	return true
}

func New(pattern string) (*Glob, error) {
	negate := strings.HasPrefix(pattern, "!")
	if negate {
		pattern = pattern[1:]
	}

	if len(pattern) == 0 {
		return nil, errors.New("glob: empty pattern")
	}
	chunks, err := parse(pattern)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, ErrInternal // WARN: remove ErrInternal
	}

	g := &Glob{
		pattern: pattern,
		negate:  negate,
	}
	if s := chunks.literal(); s != "" {
		g.literal = s
		return g, nil
	}
	if s := chunks.prefix(); s != "" {
		g.prefix = s
		return g, nil
	}
	if s := chunks.extension(); s != "" {
		g.extension = s
		return g, nil
	}
	return g, nil
}

func main() {
	r, err := parseRange("[^[a-zABC[:digit:]!@]")
	if err != nil {
		Fatal(err)
	}
	fmt.Println(r.String())
	return

	// chunks, err := parse("*.[ch]a?c")
	chunks, err := parse("\\]")
	if err != nil {
		Fatal(err)
	}
	for i, c := range chunks {
		fmt.Printf("%d: %s\n", i, c.String())
	}
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
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
