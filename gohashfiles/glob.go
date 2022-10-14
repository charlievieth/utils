package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func containsSeparator(path string) bool {
	if runtime.GOOS == "windows" {
		for i := 0; i < len(path); i++ {
			if os.IsPathSeparator(path[i]) {
				return true
			}
		}
		return false
	}
	return strings.IndexByte(path, '/') != -1
}

type Glob struct {
	pattern  string
	negate   bool
	fullPath bool
}

func (g *Glob) Pattern() string { return g.pattern }
func (g *Glob) Negated() bool   { return g.negate }

func (g Glob) String() string {
	return fmt.Sprintf("{Pattern: %q Negate: %t}", g.pattern, g.negate)
}

func NewGlob(s string) (*Glob, error) {
	if s == "" {
		return nil, errors.New("empty pattern")
	}
	negate := strings.HasPrefix(s, "!")
	if negate {
		s = strings.TrimPrefix(s, "!")
	}
	if _, err := filepath.Match(s, ""); err != nil {
		return nil, fmt.Errorf("%w: %q", err, s)
	}
	return &Glob{
		pattern:  s,
		negate:   negate,
		fullPath: containsSeparator(s),
	}, nil
}

func (g *Glob) Match(name string) bool {
	if !g.fullPath {
		name = filepath.Base(name)
	}
	ok, err := filepath.Match(g.pattern, name)
	return err == nil && ok == !g.negate
}

type GlobSet struct {
	globs []*Glob
}

func (gs *GlobSet) Set(s string) error {
	g, err := NewGlob(s)
	if err != nil {
		return err
	}
	gs.globs = append(gs.globs, g)
	return nil
}

func (g *GlobSet) Exclude(name string) bool {
	if g == nil {
		return false
	}
	for _, g := range g.globs {
		if g.Match(name) {
			return true
		}
	}
	return false
}

func (g *GlobSet) Match(name string) bool {
	if g == nil {
		return true
	}
	for _, g := range g.globs {
		if !g.Match(name) {
			return false
		}
	}
	return true
}

func (g *GlobSet) String() string {
	return fmt.Sprintf("%s", g.globs)
}

func (g *GlobSet) Type() string { return "globset" }

func (gs *GlobSet) Append(val string) error { return gs.Set(val) }
