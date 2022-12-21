package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

func AudioMD5(ctx context.Context, filename string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-i", filename, "-vn", "-f", "md5", "-")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = filepath.Dir(filename)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s: %s", err, filepath.Base(filename),
			strings.TrimSpace(stderr.String()))
	}

	out := strings.TrimSpace(stdout.String())
	algo, hash, ok := strings.Cut(out, "=")
	if !ok || algo != "MD5" {
		return "", fmt.Errorf("%s: invalid output: %q", filepath.Base(filename), out)
	}
	return hash, nil
}

type FileHash struct {
	Hash string
	Size int64
}

func main() {
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stderr)

	root, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.OpenFile("audio_hashes.json", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	nameCh := make(chan string, 512)
	hashes := make(map[string]FileHash)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var nerr int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range nameCh {
				if *verbose {
					log.Println("processing:", name)
				}
				fi, err := os.Stat(name)
				if err != nil {
					if !os.IsNotExist(err) {
						log.Printf("error: %s: %s\n", name, err)
					}
					if atomic.AddInt32(&nerr, 1) > 50 {
						log.Println("error: too many errors")
						return
					}
					continue
				}
				hash, err := AudioMD5(ctx, name)
				if err != nil {
					log.Println("error:", err)
					if atomic.AddInt32(&nerr, 1) > 50 {
						log.Println("error: too many errors")
						return
					}
					continue
				}
				mu.Lock()
				hashes[name] = FileHash{Hash: hash, Size: fi.Size()}
				mu.Unlock()
			}
		}()
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() && filepath.Ext(path) == ".mp3" {
			select {
			case nameCh <- path:
				// Ok
			case <-ctx.Done():
				log.Println("warn: context cancelled")
				return ctx.Err()
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("error: walking %s: %s\n", root, err)
		os.Exit(1)
	}
	close(nameCh)
	wg.Wait()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "    ")
	if err := enc.Encode(hashes); err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var format string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		format = "Error"
	}
	switch err.(type) {
	case error, string:
		fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
	}
	os.Exit(1)
}
