package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
)

func isBinary(buf []byte, name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return true // ignore
	}
	n, err := f.Read(buf)
	f.Close()
	if err != nil {
		return err != io.EOF // ignore if not EOF
	}
	return bytes.IndexByte(buf[:n], 0) != -1
}

func isExecutable(file string, mode os.FileMode) bool {
	if runtime.GOOS != "windows" {
		return !mode.IsDir() && mode&0111 != 0
	}
	_, err := exec.LookPath(file)
	return err == nil
}

// TODO: don't exclude supported image formats
func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case "bmp", "gif", "jpeg", "jpg", "pdf", "png", "raw", "svg", "webp":
		return true
	}
	return false
}

func readdirnames(name string) ([]string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	return names, err
}

func CompleteProject(prefix string) []string {
	dir, base := filepath.Split(prefix)
	openDir := dir
	if dir == "" {
		openDir = "."
	}
	names, err := readdirnames(openDir)
	if err != nil {
		return nil
	}
	var matches []string
	for _, s := range names {
		if strings.HasSuffix(s, ".sublime-project") && HasPrefix(s, base) {
			matches = append(matches, filepath.Join(dir, s))
		}
	}
	if len(matches) != 0 {
		sort.Strings(matches)
	}
	return matches
}

func HasPrefix(s, prefix string) bool {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return len(s) >= len(prefix) && (s[0:len(prefix)] == prefix ||
			strings.EqualFold(s[0:len(prefix)], prefix))
	}
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// Only check for binary files when there are 64 or fewer possible matches
const BinaryFileCutoff = 64

func CompleteDirs(prefix string) []string {
	dir, base := filepath.Split(prefix)
	openDir := dir
	if dir == "" {
		openDir = "."
	}
	dents, err := os.ReadDir(openDir)
	if err != nil {
		return nil
	}
	var matches []string
	for _, e := range dents {
		name := e.Name()
		if name == "" || !HasPrefix(name, base) {
			continue
		}
		mode := e.Type()
		path := filepath.Join(dir, name)
		if mode&os.ModeSymlink != 0 {
			fi, err := os.Stat(path)
			if err != nil {
				continue
			}
			mode = fi.Mode()
		}
		if mode.IsDir() {
			matches = append(matches, path+"/")
		}
	}
	return matches
}

func CompleteFiles(prefix string) []string {
	dir, base := filepath.Split(prefix)
	openDir := dir
	if dir == "" {
		openDir = "."
	}
	dents, err := os.ReadDir(openDir)
	if err != nil {
		return nil
	}
	if base != "" {
		a := dents[:0]
		for _, e := range dents {
			if name := e.Name(); name != "" && HasPrefix(name, base) {
				a = append(a, e)
			}
		}
		dents = a
	}
	if len(dents) == 0 {
		return nil
	}

	var matches []string
	buf := make([]byte, 512)
	for _, e := range dents {
		mode := e.Type()
		path := filepath.Join(dir, e.Name())
		if mode&os.ModeSymlink != 0 {
			fi, err := os.Stat(path)
			if err != nil {
				continue
			}
			mode = fi.Mode()
		}

		// if mode.IsDir() || (mode.IsRegular() && !isBinary(buf, path)) {
		if mode.IsDir() {
			matches = append(matches, path+"/")
		}

		// TODO: exclude images from the binary check
		//
		// Only check for binary files when there are fewer than
		// BinaryFileCutoff matches or the file is executable.
		if mode.IsRegular() &&
			(len(dents) > BinaryFileCutoff && !isExecutable(path, mode)) ||
			!isBinary(buf, path) {

			matches = append(matches, path)
		}
	}
	return matches
}

var subl = complete.Command{
	Args: complete.PredictFunc(CompleteFiles),
	Flags: map[string]complete.Predictor{
		"a ":                   complete.PredictFunc(CompleteDirs),
		"add":                  complete.PredictFunc(CompleteDirs),
		"w ":                   predict.Nothing,
		"wait":                 predict.Nothing,
		"b ":                   predict.Nothing,
		"background":           predict.Nothing,
		"s ":                   predict.Nothing,
		"stay":                 predict.Nothing,
		"safe-mode":            predict.Nothing,
		"h ":                   predict.Nothing,
		"help":                 predict.Nothing,
		"v ":                   predict.Nothing,
		"version":              predict.Nothing,
		"n ":                   predict.Nothing,
		"new-window":           predict.Nothing,
		"command":              predict.Nothing,
		"launch-or-new-window": predict.Nothing,
		"project":              complete.PredictFunc(CompleteProject),
	},
}

func main() {
	subl.Complete("subl")
}
