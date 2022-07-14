package main

import (
	"bytes"
	"flag"
	"fmt"
	urlpkg "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func GitBranch(wd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = wd
	b, err := cmd.CombinedOutput()
	out := string(bytes.TrimSpace(b))
	if err != nil {
		if out == "" {
			out = "no branch found"
		}
		return "", fmt.Errorf("git sha: %s: %s", err, out)
	}
	return out, nil
}

func GitSHA(wd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = wd
	b, err := cmd.CombinedOutput()
	out := string(bytes.TrimSpace(b))
	if err != nil {
		if out == "" {
			out = "no SHA found"
		}
		return "", fmt.Errorf("git sha: %s: %s", err, out)
	}
	return out, nil
}

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

// GitDir returns the directory containing the '.git' directory
func GitDir(path string) (string, error) {
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}
	vol := filepath.VolumeName(path)
	if vol == "" {
		vol = "/"
	}
	elems := strings.Split(filepath.Clean(path), string(os.PathSeparator))
	if len(elems) != 0 {
		elems[0] = vol + elems[0]
		for i := len(elems); i > 0; i-- {
			dir := filepath.Join(elems[:i]...)
			name := filepath.Join(dir, ".git")
			if isDir(name) {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("not a git directory: %s", path)
}

func IsTopLevelDir(gitdir, path string) bool {
	rel, err := filepath.Rel(gitdir, path)
	return err == nil && rel == "."
}

func TrimPathPrefix(path, prefix string) string {
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimPrefix(path, prefix)
		return strings.Trim(path, "/")
	}
	return path
}

func ConvertRemote(url string) (string, error) {
	orig := url // for errors
	if strings.HasPrefix(url, "git@") {
		host, path, ok := strings.Cut(strings.TrimPrefix(url, "git@"), ":")
		if !ok {
			return "", fmt.Errorf("invalid SSH like URL: %q", orig)
		}
		url = fmt.Sprintf("https://%s/%s", host, path)
	}
	u, err := urlpkg.Parse(url)
	if err != nil {
		return "", err
	}
	if u.Host == "go.googlesource.com" {
		u.Host = "github.com"
		u.Path = path.Join("golang", u.Path)
	}
	u.Path = strings.TrimSuffix(u.Path, ".git")
	return u.String(), nil
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

type FileArgument struct {
	Path string
	Info os.FileInfo
}

func realMain() error {
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, ".")
	}
	files := make([]FileArgument, len(args))
	for i, s := range args {
		path, err := filepath.Abs(s)
		if err != nil {
			return err
		}
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		files[i] = FileArgument{
			Path: path,
			Info: fi,
		}
	}

	openExe, err := exec.LookPath("open")
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: \"open\" command not found")
	}

	for _, file := range files {
		wd := file.Path
		if !file.Info.IsDir() {
			wd = filepath.Dir(wd)
		}
		remote, err := GitURL(wd)
		if err != nil {
			return err
		}
		url, err := ConvertRemote(remote)
		if err != nil {
			return err
		}
		dir, err := GitDir(wd)
		if err != nil {
			return err
		}
		var branch string
		if *forceMaster {
			branch = "master"
		} else {
			branch, _ = GitBranch(wd)
		}
		var sha string
		if branch == "" {
			sha, err = GitSHA(wd)
			if err != nil {
				return err
			}
		}
		repoPath := TrimPathPrefix(file.Path, dir)
		if file.Info.IsDir() {
			if branch != "" {
				// Don't use '/tree/' when opening the top-level directory
				if branch == "master" && IsTopLevelDir(dir, file.Path) {
					url = fmt.Sprintf("%s/%s", url, repoPath)
				} else {
					url = fmt.Sprintf("%s/tree/%s/%s", url, branch, repoPath)
				}
			} else {
				url = fmt.Sprintf("%s/tree/%s/%s", url, sha, repoPath)
			}
		} else {
			if branch != "" {
				url = fmt.Sprintf("%s/blob/%s/%s", url, branch, repoPath)
			} else {
				url = fmt.Sprintf("%s/blob/%s/%s", url, sha, repoPath)
			}
		}

		fmt.Println(url)
		if openExe != "" {
			out, err := exec.Command(openExe, url).CombinedOutput()
			if err != nil {
				return fmt.Errorf("open: %s", strings.TrimSpace(string(out)))
			}
		}
	}

	return nil
}

var forceMaster = flag.Bool("m", false, "use master branch")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s: [FILE]...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
