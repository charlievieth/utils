package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// remote:
// git -C "$PWD" config remote.origin.url

// default branch:
//   1. git symbolic-ref refs/remotes/origin/HEAD
//   2. git ls-remote --exit-code --symref 'https://github.com/asottile/all-repos.git' HEAD

func GetCurrentBranch(wd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = wd
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}

func GitDefaultBranch(wd, remote string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--symref", remote, "HEAD")
	cmd.Dir = wd
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ls-remote: %w", err)
	}
	s := string(bytes.TrimSpace(b))
	if !strings.HasPrefix(s, "ref: refs/heads/") {
		return "", fmt.Errorf("ls-remote output does not start with: "+
			"%q got: %q", "ref: refs/heads/", s)
	}
	if !strings.HasSuffix(s, "\tHEAD") {
		return "", fmt.Errorf("ls-remote output does not end with: "+
			"%q got: %q", "\tHEAD", s)
	}
	s = strings.TrimPrefix(s, "ref: refs/heads/")
	s = strings.TrimSuffix(s, "\tHEAD")
	return s, nil
}

func GitRemote(wd string) (string, error) {
	cmd := exec.Command("git", "config", "remote.origin.url")
	cmd.Dir = wd
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}

func GitCmd(wd string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = wd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type CmdRunner struct {
	ctx context.Context
	wd  string
	err error
}

func NewCmdRunner(ctx context.Context, wd string) *CmdRunner {
	return &CmdRunner{ctx: ctx, wd: wd}
}

func (c *CmdRunner) Err() error {
	if c.err == nil {
		return nil
	}
	return fmt.Errorf("%q: %w", c.wd, c.err)
}

func (c *CmdRunner) Git(args ...string) {
	if c.err != nil {
		return
	}
	select {
	case <-c.ctx.Done():
		c.err = c.ctx.Err()
	default:
		c.err = GitCmd(c.wd, args...)
	}
}

func FetchReset(ctx context.Context, wd string) error {
	branch, err := GetCurrentBranch(wd)
	if branch != "master" || err != nil {
		remote, err := GitRemote(wd)
		if err != nil {
			return fmt.Errorf("%q: remote: %w", wd, err)
		}
		branch, err = GitDefaultBranch(wd, remote)
	}
	if err != nil {
		return fmt.Errorf("%q: getting default branch: %w", wd, err)
	}
	if branch == "" {
		return fmt.Errorf("%q: empty default branch", wd)
	}
	cmd := NewCmdRunner(ctx, wd)
	cmd.Git("remote", "set-branches", "origin", branch)
	cmd.Git("fetch", "origin")
	cmd.Git("checkout", branch)
	cmd.Git("reset", "--hard", "origin/"+branch)
	return cmd.Err()
}

// TODO: rename
type DefaultBranchError struct {
	Remote, Current, Default string
}

func (e *DefaultBranchError) Error() string {
	return fmt.Sprintf("%s: current branch %q is not the default branch %q",
		e.Remote, e.Current, e.Default)
}

func IsDefaultBranchError(err error) bool {
	_, ok := err.(*DefaultBranchError)
	return err != nil && ok
}

func FetchResetSafe(ctx context.Context, wd string) error {
	branch, err := GetCurrentBranch(wd)
	if branch != "master" || err != nil {
		remote, err := GitRemote(wd)
		if err != nil {
			return fmt.Errorf("%q: remote: %w", wd, err)
		}
		def, err := GitDefaultBranch(wd, remote)
		if err != nil {
			return fmt.Errorf("%q: getting default branch: %w", wd, err)
		}
		if def != branch {
			return &DefaultBranchError{remote, branch, def}
		}
	}
	if err != nil {
		return fmt.Errorf("%q: getting default branch: %w", wd, err)
	}
	if branch == "" {
		return fmt.Errorf("%q: empty default branch", wd)
	}
	cmd := NewCmdRunner(ctx, wd)
	cmd.Git("remote", "set-branches", "origin", branch)
	cmd.Git("fetch", "origin")
	cmd.Git("checkout", branch)
	cmd.Git("reset", "--hard", "origin/"+branch)
	return cmd.Err()
}

func UpdateWorker(ctx context.Context, wg *sync.WaitGroup, repos chan string) {
	defer wg.Done()
	for repo := range repos {
		fmt.Fprintf(os.Stderr, "# Updating: %s\n", filepath.Base(repo))
		if err := FetchResetSafe(ctx, repo); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func isDir(name string) bool {
	fi, err := os.Lstat(name)
	return err == nil && fi.IsDir()
}

func realMain(root string) error {
	const numWorkers = 6

	repoCh := make(chan string, numWorkers)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ch := make(chan os.Signal, 8)
		signal.Notify(ch, os.Interrupt)
		for {
			select {
			case sig := <-ch:
				fmt.Fprintf(os.Stderr, "Signal %q: stopping\n", sig)
				cancel()
				// drain repoCh
			drainLoop:
				for {
					select {
					case <-repoCh:
					default:
						break drainLoop
					}
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go UpdateWorker(ctx, &wg, repoCh)
	}

	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() && isDir(filepath.Join(path, ".git")) {
			select {
			case repoCh <- path:
				// Ok
			case <-ctx.Done():
				return ctx.Err()
			}
			return filepath.SkipDir
		}
		return nil
	})

	wg.Wait()
	return err
}

func realMainRandomWalk(root string) error {
	const numWorkers = 6

	var repos []string
	repoCh := make(chan string, numWorkers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 8)
		signal.Notify(ch, os.Interrupt)
		defer signal.Reset(os.Interrupt)
		for {
			select {
			case sig := <-ch:
				fmt.Fprintf(os.Stderr, "Signal %q: stopping\n", sig)
				cancel()
				for {
					select {
					case <-repoCh:
					default:
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	start := time.Now()
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() && isDir(filepath.Join(path, ".git")) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				repos = append(repos, path)
			}
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "# Found %d repos in %s\n", len(repos), time.Since(start))

	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	rr.Shuffle(len(repos), func(i, j int) {
		repos[i], repos[j] = repos[j], repos[i]
	})

	start = time.Now()
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go UpdateWorker(ctx, &wg, repoCh)
	}

	updated := 0
Loop:
	for _, repo := range repos {
		select {
		case repoCh <- repo:
			updated++
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Context canceled (%s): stopping\n", ctx.Err())
			break Loop
		}
	}
	fmt.Fprintln(os.Stderr, "# Waiting for workers to stop")
	close(repoCh)
	wg.Wait()

	dur := time.Since(start)
	fmt.Fprintf(os.Stderr, "# Updated %d repos in %s %s/repo\n",
		updated, dur, dur/time.Duration(updated))
	return ctx.Err()
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s: DIRECTORY...\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	for i := 1; i < len(os.Args); i++ {
		root := os.Args[i]
		if err := realMainRandomWalk(root); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
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

/*
func GitURL(wd string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = wd
	b, err := cmd.CombinedOutput()
	out := string(bytes.TrimSpace(b))
	if err != nil {
		if out == "" {
			out = "no remote found"
		}
		return "", fmt.Errorf("git url: %s: %s", err, out)
	}
	return out, nil
}
*/
