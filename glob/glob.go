package glob

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var (
	_ flag.Value  = (*Glob)(nil)
	_ flag.Getter = (*Glob)(nil)
	_ pflag.Value = (*Glob)(nil)

	_ flag.Value       = (*GlobSet)(nil)
	_ flag.Getter      = (*GlobSet)(nil)
	_ pflag.Value      = (*GlobSet)(nil)
	_ pflag.SliceValue = (*GlobSet)(nil)
)

// MatchKind is the kind of match used by Globs.
type MatchKind int32

const (
	MatchGlob     MatchKind = iota // match with filepath.Glob (`*.[ch]`)
	MatchPrefix                    // match only prefix (`main.*`)
	MatchSuffix                    // match only suffix (`*.go`)
	MatchContains                  // match if path contains pattern (`*main*`)
	MatchExact                     // exact match (`main.go`)
)

var matchKindStrings = [...]string{
	MatchGlob:     "MatchGlob",
	MatchPrefix:   "MatchPrefix",
	MatchSuffix:   "MatchSuffix",
	MatchContains: "MatchContains",
	MatchExact:    "MatchExact",
}

func (m MatchKind) String() string {
	if uint(m) <= uint(len(matchKindStrings)) {
		return matchKindStrings[m]
	}
	return fmt.Sprintf("MatchKind(%d)", m)
}

// A Glob stores a shell filename pattern for matching.
type Glob struct {
	pattern string
	kind    MatchKind
	negated bool
}

// Parse parses shell filename pattern and returns a new Glob or an
// error if the pattern is invalid. Precede the pattern with '!' to
// negate the Glob.
//
//	Parse("*.go")  // Matches any string ending with ".go"
//	Parse("!*.go") // Matches any string *not* ending with ".go"
func Parse(pattern string) (*Glob, error) {
	negated := strings.HasPrefix(pattern, "!")
	if negated {
		pattern = strings.TrimPrefix(pattern, "!")
	}
	if pattern == "" {
		return nil, errors.New("glob: empty glob pattern")
	}
	var kind MatchKind
	switch {
	case strings.HasPrefix(pattern, "*") && !hasMeta(pattern[1:]):
		kind = MatchSuffix
		pattern = pattern[1:]
	case strings.HasSuffix(pattern, "*") && !hasMeta(pattern[:len(pattern)-1]):
		kind = MatchPrefix
		pattern = pattern[:len(pattern)-1]
	case strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") &&
		!hasMeta(pattern[1:len(pattern)-1]):
		kind = MatchContains
		pattern = pattern[1 : len(pattern)-1]
	case !hasMeta(pattern):
		kind = MatchExact
	default:
		kind = MatchGlob
		if _, err := filepath.Match(pattern, pattern); err != nil {
			return nil, fmt.Errorf("glob: %w: %q", err, pattern)
		}
	}
	return &Glob{pattern, kind, negated}, nil
}

// Kind returns the Glob's MatchKind.
func (g *Glob) Kind() MatchKind { return g.kind }

// Negated returns if the Glob is negated.
func (g *Glob) Negated() bool { return g.negated }

// Get returns Glob *g and exists to satisfy the flag.Getter interface.
func (g *Glob) Get() any { return g }

// Pattern returns the pattern the Glob was created with.
func (g *Glob) Pattern() string {
	var s string
	switch g.kind {
	case MatchGlob:
		s = g.pattern
	case MatchPrefix:
		s = g.pattern + "*"
	case MatchSuffix:
		s = "*" + g.pattern
	case MatchContains:
		s = "*" + g.pattern + "*"
	case MatchExact:
		s = g.pattern
	}
	if g.negated {
		s = "!" + s
	}
	return s
}

func (g *Glob) String() string {
	return g.Pattern()
}

func (g *Glob) match(path string) bool {
	var match bool
	switch g.kind {
	case MatchGlob:
		match, _ = filepath.Match(g.pattern, path)
	case MatchPrefix:
		match = strings.HasPrefix(path, g.pattern) ||
			strings.HasPrefix(filepath.Base(path), g.pattern)
	case MatchSuffix:
		match = strings.HasSuffix(path, g.pattern) ||
			strings.HasSuffix(filepath.Base(path), g.pattern)
	case MatchContains:
		match = strings.Contains(filepath.Base(path), g.pattern)
	case MatchExact:
		match = (path == g.pattern || filepath.Base(path) == g.pattern)
	}
	return match
}

// Match returns if the Glob matches path.
func (g *Glob) Match(path string) bool {
	match := g.match(path)
	return match == !g.negated
}

// Sets the Glob's pattern to pattern and exists to implement the [flag.Value]
// and [pflag.Value] interfaces.
func (g *Glob) Set(pattern string) error {
	gg, err := Parse(pattern)
	if err != nil {
		return err
	}
	*g = *gg
	return nil
}

// Type implements [pflag.Value].
func (g *Glob) Type() string { return "Glob" }

// TODO: "GlobSet" stutters with package name - rename.

// A GlobSet is an ordered set of Globs. When matching the Glob given later
// takes precedence.
type GlobSet struct {
	globs []*Glob
}

// NewGlobSet returns a new GlobSet.
func NewGlobSet(patterns ...string) (*GlobSet, error) {
	globs := make([]*Glob, len(patterns))
	for i, p := range patterns {
		g, err := Parse(p)
		if err != nil {
			return nil, err
		}
		globs[i] = g
	}
	return &GlobSet{globs}, nil
}

// Add appends Glob g to the GlobSet.
func (s *GlobSet) Add(g *Glob) {
	s.globs = append(s.globs, g)
}

// Parse parses shell filename pattern and appends it to the GlobSet.
func (s *GlobSet) Parse(pattern string) error {
	g, err := Parse(pattern)
	if err != nil {
		return err
	}
	s.Add(g)
	return nil
}

// Match returns if the GlobSet matches path. If multiple Globs match path,
// the glob given later takes precedence.
func (s *GlobSet) Match(path string) bool {
	// test in reverse order (matches `rg`)
	for i := len(s.globs) - 1; i >= 0; i-- {
		if s.globs[i].match(path) {
			return !s.globs[i].negated
		}
	}
	return false
}

// Get returns GlobSet *s and exists to satisfy the flag.Getter interface.
func (s *GlobSet) Get() any { return s }

// Sets the GlobSet's pattern to pattern, removing any existing patterns and
// exists to implement the [flag.Value] and [pflag.Value] interfaces.
func (s *GlobSet) Set(pattern string) error {
	g, err := Parse(pattern)
	if err != nil {
		return err
	}
	s.globs = []*Glob{g}
	return nil
}

func (s *GlobSet) String() string {
	a := s.GetSlice()
	for i := range a {
		a[i] = strconv.Quote(a[i])
	}
	return "[" + strings.Join(a, ", ") + "]"
}

// Type implements [pflag.Value].
func (s *GlobSet) Type() string { return "GlobSet" }

// Append appends pattern to the GlobSet and exists to implement the
// [pflag.SliceValue] interface.
func (s *GlobSet) Append(pattern string) error {
	return s.Parse(pattern)
}

// Replace replaces all the GlobSet's patterns and exists to implement the
// [pflag.SliceValue] interface.
func (s *GlobSet) Replace(patterns []string) error {
	for i := range s.globs {
		s.globs[i] = nil
	}
	s.globs = s.globs[:0]
	for _, p := range patterns {
		g, err := Parse(p)
		if err != nil {
			return err
		}
		s.globs = append(s.globs, g)
	}
	return nil
}

// GetSlice returns a list of the GlobSet's patterns and exists to implement the
// [pflag.SliceValue] interface.
func (s *GlobSet) GetSlice() []string {
	a := make([]string, len(s.globs))
	for i, g := range s.globs {
		a[i] = g.Pattern()
	}
	return a
}

// hasMeta returns true if pattern contains any glob meta characters.
func hasMeta(pattern string) bool {
	// Include "/" so that we fallback to using filepath.Glob
	if filepath.Separator == '/' {
		return strings.ContainsAny(pattern, `*?[]/`)
	}
	return strings.ContainsAny(pattern, `*?[]/`+string(filepath.Separator))
}
