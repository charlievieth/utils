package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"
)

type Map struct {
	b []byte // buffer
}

func (m *Map) MapBytes(mapping func(r rune) rune, s []byte) []byte {
	// In the worst case, the slice can grow when mapped, making
	// things unpleasant. But it's so rare we barge in assuming it's
	// fine. It could also shrink but that falls out naturally.
	maxbytes := len(s) // length of b
	nbytes := 0        // number of bytes encoded in b
	b := m.b
	if len(b) < maxbytes {
		b = make([]byte, maxbytes)
	}
	for i := 0; i < len(s); {
		wid := 1
		r := rune(s[i])
		if r >= utf8.RuneSelf {
			r, wid = utf8.DecodeRune(s[i:])
		}
		r = mapping(r)
		if r >= 0 {
			rl := utf8.RuneLen(r)
			if rl < 0 {
				rl = len(string(utf8.RuneError))
			}
			if nbytes+rl > maxbytes {
				// Grow the buffer.
				maxbytes = maxbytes*2 + utf8.UTFMax
				nb := make([]byte, maxbytes)
				copy(nb, b[0:nbytes])
				b = nb
			}
			nbytes += utf8.EncodeRune(b[nbytes:maxbytes], r)
		}
		i += wid
	}
	m.b = b
	return b[0:nbytes]
}

func (m *Map) ToUpperBytes(s []byte) []byte {
	isASCII, hasLower := true, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasLower = hasLower || (c >= 'a' && c <= 'z')
	}

	if isASCII { // optimize for ASCII-only strings.
		if !hasLower {
			return s
		}
		b := m.b
		if len(b) < len(s) {
			b = make([]byte, len(s))
		}
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
			b[i] = c
		}
		m.b = b
		return b[:len(s)]
	}
	return m.MapBytes(unicode.ToUpper, s)
}

func (m *Map) MapString(mapping func(rune) rune, s string) []byte {
	// In the worst case, the string can grow when mapped, making
	// things unpleasant. But it's so rare we barge in assuming it's
	// fine. It could also shrink but that falls out naturally.

	// The output buffer b is initialized on demand, the first
	// time a character differs.
	b := m.b
	// nbytes is the number of bytes encoded in b.
	var nbytes int

	for i, c := range s {
		r := mapping(c)
		if r == c {
			continue
		}

		if n := len(s) + utf8.UTFMax; len(b) < n {
			b = make([]byte, n)
		}
		nbytes = copy(b, s[:i])
		if r >= 0 {
			if r <= utf8.RuneSelf {
				b[nbytes] = byte(r)
				nbytes++
			} else {
				nbytes += utf8.EncodeRune(b[nbytes:], r)
			}
		}

		if c == utf8.RuneError {
			// RuneError is the result of either decoding
			// an invalid sequence or '\uFFFD'. Determine
			// the correct number of bytes we need to advance.
			_, w := utf8.DecodeRuneInString(s[i:])
			i += w
		} else {
			i += utf8.RuneLen(c)
		}

		s = s[i:]
		break
	}

	if b == nil {
		if len(b) < len(s) {
			b = make([]byte, len(s))
		}
		copy(b, s)
		m.b = b
		return b[:len(s)]
	}

	for _, c := range s {
		r := mapping(c)

		// common case
		if (0 <= r && r <= utf8.RuneSelf) && nbytes < len(b) {
			b[nbytes] = byte(r)
			nbytes++
			continue
		}

		// b is not big enough or r is not a ASCII rune.
		if r >= 0 {
			if nbytes+utf8.UTFMax >= len(b) {
				// Grow the buffer.
				nb := make([]byte, 2*len(b))
				copy(nb, b[:nbytes])
				b = nb
			}
			nbytes += utf8.EncodeRune(b[nbytes:], r)
		}
	}

	m.b = b
	return b[:nbytes]
}

func (m *Map) ToUpperString(s string) []byte {
	isASCII, hasLower := true, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasLower = hasLower || (c >= 'a' && c <= 'z')
	}

	if isASCII { // optimize for ASCII-only strings.
		b := m.b
		if len(b) < len(s) {
			b = make([]byte, len(s))
		}
		if !hasLower {
			copy(b, s)
			goto Exit
		}
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
			b[i] = c
		}
	Exit:
		m.b = b
		return b[:len(s)]
	}
	return m.MapString(unicode.ToUpper, s)
}

var errNegativeRead = errors.New("toupper: reader returned negative count from Read")

func readStdin(buf []byte) (int, error) {
	const maxConsecutiveEmptyReads = 100
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := os.Stdin.Read(buf)
		if n < 0 {
			panic(errNegativeRead)
		}
		if n > 0 || err != nil {
			return n, err
		}
	}
	return 0, io.ErrNoProgress
}

func lastRune(buf []byte) int {
	n := len(buf)
	i := n - 1
	for ; i >= 0 && buf[i] >= utf8.RuneSelf; i-- {
	}
	return n - i - 1
}

func lastAsciiIndex(buf []byte) int {
	i := len(buf) - 1
	for ; i >= 0 && buf[i] >= utf8.RuneSelf; i-- {
	}
	return i
}

// WARN (CEV): this thing is borked
func parseStdin() (err error) {
	var m Map
	var keep []byte
	buf := make([]byte, 2)
	off := 0
	for {
		if off = len(keep); off != 0 {
			// copy(buf, keep)
			buf = append(buf[:0], keep...)
			keep = keep[:0]
		}
		// n, e := readStdin(buf[off:])
		n, e := os.Stdin.Read(buf[off:])
		n += off

		// WARN (CEV): trying this approach
		// if i := lastAsciiIndex(buf[:n]); i < n {
		// 	buf = buf[:n]
		// }

		if sz := lastRune(buf[:n]); sz > 0 {
			// fmt.Println(" XX ")
			// fmt.Println("n:", n, "off:", off)
			// fmt.Println("sz:", sz)
			// fmt.Println("len:", len(buf))
			keep = buf[:n-sz]
			// buf = buf[n-sz:]
			n -= sz
			fmt.Println("\nKEEP:", len(keep))
		}
		o := m.ToUpperBytes(buf[:n])
		if _, err := os.Stdout.Write(o); err != nil {
			return err
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	return err
}

func parseArgs(args []string) error {
	var m Map
	for _, s := range args {
		m.b = append(m.ToUpperString(s), '\n')
		if _, err := os.Stdout.Write(m.b); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var err error
	if len(os.Args) > 1 {
		err = parseArgs(os.Args[1:])
	} else {
		err = parseStdin()
	}
	if err != nil {
		os.Stderr.WriteString("error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

// func Fatal(err interface{}) {
// 	if err == nil {
// 		return
// 	}
// 	var s string
// 	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
// 		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
// 	} else {
// 		s = "Error"
// 	}
// 	switch err.(type) {
// 	case error, string, fmt.Stringer:
// 		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
// 	default:
// 		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
// 	}
// 	os.Exit(1)
// }
