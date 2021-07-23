package token

//go:generate stringer -type=Token

type Token int8

const (
	None Token = iota
	// TODO: do we need a PosixWord Token ???
	Word
	Whitespace
	Quote
	Escape
	// EscapedQuote // TODO: do we need this ?
	Punctuation
	Comment
)

var tokens = [128]Token{
	'#':  Comment,
	'\'': Quote, '"': Quote, // WARN: '"' is also an EscapedQuote
	'\\': Escape,

	' ': Whitespace, '\t': Whitespace, '\r': Whitespace, '\n': Whitespace,

	'(': Punctuation, ')': Punctuation, ';': Punctuation, '<': Punctuation,
	'>': Punctuation, '|': Punctuation, '&': Punctuation,

	'a': Word, 'b': Word, 'c': Word, 'd': Word, 'f': Word, 'e': Word, 'g': Word,
	'h': Word, 'i': Word, 'j': Word, 'k': Word, 'l': Word, 'm': Word, 'n': Word,
	'o': Word, 'p': Word, 'q': Word, 'r': Word, 's': Word, 't': Word, 'u': Word,
	'v': Word, 'w': Word, 'x': Word, 'y': Word, 'z': Word, 'A': Word, 'B': Word,
	'C': Word, 'D': Word, 'E': Word, 'F': Word, 'G': Word, 'H': Word, 'I': Word,
	'J': Word, 'K': Word, 'L': Word, 'M': Word, 'N': Word, 'O': Word, 'P': Word,
	'Q': Word, 'R': Word, 'S': Word, 'T': Word, 'U': Word, 'V': Word, 'W': Word,
	'X': Word, 'Y': Word, 'Z': Word, '0': Word, '1': Word, '2': Word, '3': Word,
	'4': Word, '5': Word, '6': Word, '7': Word, '8': Word, '9': Word, '_': Word,
}

func IsComment(r rune) bool      { return r == '#' }
func IsEscape(r rune) bool       { return r == '\\' }
func IsQuote(r rune) bool        { return r == '\'' || r == '"' }
func IsEscapedQuote(r rune) bool { return r == '"' }

func IsWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func IsPunctuation(r rune) bool {
	switch r {
	case '(', ')', ';', '<', '>', '|', '&':
		return true
	}
	return false
}

func isPosixWordChar(r rune) bool {
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

func IsPosixWord(r rune) bool {
	if uint32(r) < uint32(len(tokens)) {
		return tokens[r] == Word
	}
	return isPosixWordChar(r)
}

func IsWord(r rune) bool {
	return uint32(r) < uint32(len(tokens)) && tokens[r] == Word
}

func Classify(r rune) Token {
	if uint32(r) < uint32(len(tokens)) {
		return tokens[r]
	}
	// TODO: do we need a PosixWord Token ???
	if isPosixWordChar(r) {
		return Word
	}
	return None
}
