package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/blang/semver"
	"github.com/manifoldco/promptui"
)

type Selector interface {
	Name() string
	CmdArgs() []string
	ExtraSearchPaths() []string
	ParseVersion(s string) (string, error)
}

type ParseFunc func(cmdOutput string) (string, error)

type Config struct {
	Name string
	// AltNames []string // use for alt names (consider a regex)

	// consider allowing multiple sets of args for diff versions
	CmdArgs []string

	ExtraSearchPaths []string

	ParseVersion ParseFunc
	Sort         func(versions []string) []string
}

func DefaultVersionSort(versions []string) []string {
	sort.Strings(versions)
	return versions
}

func DefaultParseVersion(s string) (string, error) {
	return strings.TrimSpace(s), nil
}

type Executable struct {
	Path, Version string
	Current       bool
}

func expandPath(s string) (string, error) {
	if strings.HasPrefix(s, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		s = home + "/" + s[1:]
	}
	if strings.Contains(s, "$") {
		s = os.ExpandEnv(s)
	}
	return filepath.Clean(s), nil
}

func Find(conf *Config) ([]Executable, error) {
	// TODO: remove duplicates
	var exes []Executable
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if p, err := exec.LookPath(dir + "/" + conf.Name); err == nil {
			exes = append(exes, Executable{Path: p, Current: len(exes) == 0})
		}
	}
	for _, extra := range conf.ExtraSearchPaths {
		exp, err := expandPath(extra)
		if err != nil {
			return nil, err
		}
		dirs, err := filepath.Glob(exp)
		if err != nil {
			return nil, err
		}
		for _, dir := range dirs {
			if p, err := exec.LookPath(dir + "/" + conf.Name); err == nil {
				exes = append(exes, Executable{Path: p, Current: len(exes) == 0})
			}
		}
	}
	return exes, nil
}

func GetVersion(conf *Config, exes []Executable) ([]Executable, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var first atomicError
	gate := make(chan struct{}, 8)
	for i := range exes {
		wg.Add(1)
		gate <- struct{}{}
		go func(e *Executable) {
			defer func() { wg.Done(); <-gate }()
			out, err := exec.CommandContext(ctx, e.Path, conf.CmdArgs...).CombinedOutput()
			if err != nil {
				first.Set(err)
				return
			}
			ver, err := conf.ParseVersion(string(bytes.TrimSpace(out)))
			if err != nil {
				first.Set(err)
				return
			}
			e.Version = ver
		}(&exes[i])
	}
	wg.Wait()
	if err := first.Err(); err != nil {
		return nil, err
	}
	return exes, nil
}

func cachDir() (string, error) {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("neither $XDG_CACHE_HOME nor $HOME are defined")
		}
		dir += "/.cache"
	}
	return filepath.Join(dir, "selectexe"), nil
}

func configDir() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
		}
		dir += "/.config"
	}
	return filepath.Join(dir, "selectexe"), nil
}

func pathsEqual(p1, p2 string) bool {
	if p1 == p2 {
		return true
	}
	s1, err := filepath.EvalSymlinks(p1)
	if err != nil {
		s1 = filepath.Clean(p1)
	}
	s2, err := filepath.EvalSymlinks(p2)
	if err != nil {
		s2 = filepath.Clean(p2)
	}
	rel, err := filepath.Rel(s1, s2)
	return err == nil && rel == "."
}

func onPath(dirname string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if pathsEqual(dirname, p) {
			return true
		}
	}
	return false
}

type CacheError struct {
	Cache string
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("selectexe cache (%q) is not on the $PATH", e.Cache)
}

func doInit() error {
	conf, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(conf, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	cache, err := cachDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cache, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	if !onPath(cache) {
		return &CacheError{cache}
	}
	return nil
}

func Run(conf *Config) error {
	if err := doInit(); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return nil
}

type atomicError struct {
	mu  sync.Mutex
	err error
}

func (e *atomicError) Set(err error) {
	if err != nil {
		e.mu.Lock()
		if e.err == nil {
			e.err = err
		}
		e.mu.Unlock()
	}
}

func (e *atomicError) Err() error {
	e.mu.Lock()
	err := e.err
	e.mu.Unlock()
	return err
}

func main() {

	sortVersions := func(versions []string) []string {
		var svs []semver.Version
		for _, s := range versions {
			v, err := semver.Parse(s)
			if err != nil {
				v = semver.Version{Build: []string{s}}
			}
			svs = append(svs, v)
		}
		return versions
	}

	parseVersion := func(out string) (string, error) {
		for _, s := range strings.Split(out, "\n") {
			if strings.HasPrefix(s, "Build Tag:") {
				return strings.TrimSpace(strings.TrimPrefix(s, "Build Tag:")), nil
			}
		}
		return "", errors.New("no version")
	}
	conf := Config{
		Name:             "cockroach",
		ExtraSearchPaths: []string{"~/.cache/cockroachdb/*"},
		CmdArgs:          []string{"version"},
		ParseVersion:     parseVersion,
		Sort:             sortVersions,
	}
	a, err := Find(&conf)
	if err != nil {
		Fatal(err)
	}
	exes, err := GetVersion(&conf, a)
	if err != nil {
		Fatal(err)
	}

	versions := make([]string, 0, len(exes))
	for _, e := range exes {
		// fmt.Println(e.Version)
		versions = append(versions, e.Version)
	}
	size := len(versions)
	if size > 10 {
		size = 10
	}
	// '{{ . | cyan }}'

	prompt := promptui.Select{
		Label: "Version",
		Items: versions,
		Size:  size,
		Templates: &promptui.SelectTemplates{
			Active: "{{ . | cyan }}",
		},
	}
	_, result, err := prompt.Run()
	if err != nil {
		Fatal(err)
	}
	fmt.Println("result:", result)
}

/*
func main() {
	const out = `
Build Tag:        v21.2.0-alpha.00000000-1761-gf661758704-dirty
Build Time:       2021/06/24 21:34:19
Distribution:     CCL
Platform:         darwin amd64 (x86_64-apple-darwin20.5.0)
Go Version:       go1.16.5
C Compiler:       Apple LLVM 12.0.5 (clang-1205.0.22.9)
Build Commit ID:  f661758704beedb762ad9e2872d7f68d1bbb9509
Build Type:       development
`

	re := regexp.MustCompile(`(?m)(?:Build Tag:\s+)(v\d+\.\d+\.\d+(?:-.*)|unknown)$`)

	for _, s := range re.FindAllStringSubmatch(out, -1) {
		fmt.Printf("%q\n", s)
	}
}
*/

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
	}
}

func PrintJSONColor(v interface{}) {
	cmd := exec.Command("jq", "--color-output", "--indent", "4")
	wc, err := cmd.StdinPipe()
	if err != nil {
		Fatal(err)
	}
	var out bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		Fatal(err)
	}

	enc := json.NewEncoder(wc)
	enc.SetIndent("", "    ")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
	}
	if err := wc.Close(); err != nil {
		Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		Fatal(fmt.Sprintf("error: jq: %s\n### OUTPUT\n%s\n##", err, bytes.TrimSpace(out.Bytes())))
	}
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var s string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		s = "Error"
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
	}
	os.Exit(1)
}
