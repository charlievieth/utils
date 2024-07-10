package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"golang.org/x/perf/benchfmt"
)

// WARN: dev only
var _ = benchfmt.Result{}

func init() {
	log.SetFlags(log.Lshortfile)
}

// type File struct {
// 	Name   string
// 	Path   string
// 	Status string // TODO: use an enum
// 	Data   string
// }

// type RepoState struct {
// }

type Benchmark struct {
	Dir        string
	ImportPath string   // import path
	Args       []string // benchmark args
	Passed     bool
	CreatedAt  time.Time

	Result    *Result
	RawResult string
}

type Config struct {
	Key   string
	Value string
	File  bool // Set if this is a file configuration key, otherwise internal
}

func NewConfig(c *benchfmt.Config) *Config {
	return &Config{
		Key:   c.Key,
		Value: string(c.Value),
		File:  c.File,
	}
}

func NewConfigs(confs ...benchfmt.Config) []Config {
	if len(confs) == 0 {
		return nil
	}
	cs := make([]Config, len(confs))
	for i, c := range confs {
		cs[i] = Config{
			Key:   c.Key,
			Value: string(c.Value),
			File:  c.File,
		}
	}
	return cs
}

type Result struct {
	// Config is the set of key/value configuration pairs for this result,
	// including file and internal configuration. This does not include
	// sub-name configuration.
	//
	// This slice is mutable, as are the values in the slice.
	// Result internally maintains an index of the keys of this slice,
	// so callers must use SetConfig to add or delete keys,
	// but may modify values in place. There is one exception to this:
	// for convenience, new Results can be initialized directly,
	// e.g., using a struct literal.
	//
	// SetConfig appends new keys to this slice and updates existing ones
	// in place. To delete a key, it swaps the deleted key with
	// the final slice element. This way, the order of these keys is
	// deterministic.
	Config []Config

	// Name is the full name of this benchmark, including all
	// sub-benchmark configuration.
	Name string

	// Iters is the number of iterations this benchmark's results
	// were averaged over.
	Iters int

	// Values is this benchmark's measurements and their units.
	Values []benchfmt.Value
}

func NewResult(r *benchfmt.Result) *Result {
	return &Result{
		Config: NewConfigs(r.Config...),
		Name:   r.Name.String(),
		Iters:  r.Iters,
		Values: append([]benchfmt.Value(nil), r.Values...),
	}
}

type FileStatus struct {
	Name   string
	Status string // TODO
	Data   string
}

type Status struct {
	// # branch.oid <commit> | (initial)        Current commit.
	// # branch.head <branch> | (detached)      Current branch.
	// # branch.upstream <upstream-branch>      If upstream is set.
	// # branch.ab +<ahead> -<behind>           If upstream is set and
	BranchOID      string
	BranchHead     string
	BranchUpstream string
	BranchAB       string
	Stash          int
}

func GitStatus(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v2", "--show-stash", "--branch")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command %q exited with error: %v: %s",
			cmd.Args, err, bytes.TrimSpace(out))
	}
	return "", nil
}

// List all go pkgs and their deps: `go list -json -deps`

func main() {
	f, err := os.Open("testdata/all.base.10.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	br := benchfmt.NewReader(f, "testdata/all.base.10.txt")
	for br.Scan() {
		r := br.Result()
		if r == nil {
			break
		}
		switch v := r.(type) {
		case *benchfmt.Result:
			PrintJSON(NewResult(v))
		case *benchfmt.UnitMetadata:
			PrintJSON(v)
		case *benchfmt.SyntaxError:
			log.Fatal("SyntaxError:", err)
		}
	}
	if err := br.Err(); err != nil {
		log.Fatal(err)
	}
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if err := enc.Encode(v); err != nil {
		log.Output(2, err.Error())
	}
}
