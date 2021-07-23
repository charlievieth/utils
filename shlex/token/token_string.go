// Code generated by "stringer -type=Token"; DO NOT EDIT.

package token

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[None-0]
	_ = x[Word-1]
	_ = x[Whitespace-2]
	_ = x[Quote-3]
	_ = x[Escape-4]
	_ = x[Punctuation-5]
	_ = x[Comment-6]
}

const _Token_name = "NoneWordWhitespaceQuoteEscapePunctuationComment"

var _Token_index = [...]uint8{0, 4, 8, 18, 23, 29, 40, 47}

func (i Token) String() string {
	if i < 0 || i >= Token(len(_Token_index)-1) {
		return "Token(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Token_name[_Token_index[i]:_Token_index[i+1]]
}
