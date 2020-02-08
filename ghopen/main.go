package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
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

func TrimPathPrefix(path, prefix string) string {
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimPrefix(path, prefix)
		return strings.Trim(path, "/")
	}
	return path
}

func ConvertRemote(url string) (string, error) {
	if strings.HasSuffix(url, ".git") {
		url = strings.TrimSuffix(url, ".git")
	}
	if strings.HasPrefix(url, "https://github.com/") {
		return url, nil
	}
	if strings.HasPrefix(url, "git@github.com:") {
		s := strings.TrimPrefix(url, "git@github.com:")
		return "https://github.com/" + s, nil
	}
	if strings.HasPrefix(url, "https://go.googlesource.com/") {
		s := strings.TrimPrefix(url, "https://go.googlesource.com/")
		return "https://github.com/golang/" + s, nil
	}
	return "", fmt.Errorf("unknown URL: %s", url)
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
		sha, err := GitSHA(wd)
		if err != nil {
			return err
		}
		dir, err := GitDir(wd)
		if err != nil {
			return err
		}
		branch, _ := GitBranch(wd)
		repoPath := TrimPathPrefix(file.Path, dir)
		if file.Info.IsDir() {
			if branch != "" {
				url = fmt.Sprintf("%s/tree/%s/%s", url, branch, repoPath)
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
		out, err := exec.Command("open", url).CombinedOutput()
		if err != nil {
			return fmt.Errorf("open: %s", strings.TrimSpace(string(out)))
		}
	}

	return nil
}

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
