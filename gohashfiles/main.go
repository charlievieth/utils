package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"text/tabwriter"
)

type FileHash struct {
	Name, Hash string
}

type FileSet struct {
	Hash  string
	Names []string
}

type FileList struct {
	mu    sync.Mutex
	files []FileHash
}

func (f *FileList) Add(h FileHash) {
	f.mu.Lock()
	f.files = append(f.files, h)
	f.mu.Unlock()
}

func (f *FileList) Files() []FileHash {
	return f.files
}

type Worker struct {
	buf   []byte
	h     hash.Hash
	files *FileList
}

func (w *Worker) HashFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	if w.buf == nil {
		w.buf = make([]byte, 32*1024)
	}
	if w.h == nil {
		w.h = sha256.New()
	}
	w.h.Reset()
	_, err = io.CopyBuffer(w.h, f, w.buf)
	f.Close()
	if err != nil {
		return err
	}
	w.files.Add(FileHash{
		Name: name,
		Hash: hex.EncodeToString(w.h.Sum(nil)),
	})
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: PATHs...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: missing PATH")
		flag.Usage()
		os.Exit(1)
	}

	numCPU := runtime.NumCPU()
	if numCPU < 4 {
		numCPU = 4
	}

	wg := new(sync.WaitGroup)
	list := new(FileList)
	workCh := make(chan string, numCPU*4)

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := &Worker{files: list}
			for name := range workCh {
				if err := w.HashFile(name); err != nil {
					fmt.Fprintf(os.Stderr, "error: %s: %v\n", name, err)
				}
			}
		}()
	}

	// type WalkFunc func(path string, info os.FileInfo, err error) error
	for _, path := range flag.Args() {
		fmt.Println("Walking:", path)
		err := filepath.Walk(path, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.Mode().IsRegular() {
				workCh <- path
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: walking path: %s: %v\n", path, err)
		}
	}
	close(workCh)
	wg.Wait()

	m := make(map[string][]string, len(list.Files()))
	for _, h := range list.Files() {
		m[h.Hash] = append(m[h.Hash], h.Name)
	}
	// for k, v := range m {
	// 	sort.Strings(v)
	// 	m[k] = v
	// }

	w := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)
	for k, v := range m {
		if len(v) > 1 {
			sort.Strings(v)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\n", len(v), k, v)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "error: flush: %v\n", err)
		os.Exit(1)
	}
}

/*
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	b := make([]byte, 0, 128)
	for _, l := range lines {
		b = b[:0]
		b = strconv.AppendInt(b, int64(l.N), 10)
		b = append(b, ':')
		b = append(b, '\t')
		b = append(b, l.S...)
		b = append(b, '\n')
		if _, err := w.Write(b); err != nil {
			Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		Fatal(err)
	}
*/

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
