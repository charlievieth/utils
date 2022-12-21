package main

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	_ "github.com/mattn/go-sqlite3"
)

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stderr)
}

type FileHash struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

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

func connectionString(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	v := u.Query()
	v.Set("_cache_size", "-4000")
	v.Set("_mutex", "full")
	u.RawQuery = v.Encode()
	return u.String(), nil
}

//go:embed sql/create_audio_files_table.sql
var CreateAudioFilesTableStmt string

//go:embed sql/insert_audio_file_stmt.sql
var InsertAudioFileStmt string

// TODO: use sqlite
func OpenDatabase(ctx context.Context, filename string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_cache_size=-4000&_mutex=full", filename)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if _, err := db.Exec(CreateAudioFilesTableStmt); err != nil {
		return nil, err
	}
	return db, nil
}

func BulkInsertHashes(ctx context.Context, db *sql.DB, hashes map[string]FileHash) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	exit := func(err error) error {
		tx.Rollback()
		return err
	}
	stmt, err := tx.PrepareContext(ctx, InsertAudioFileStmt)
	if err != nil {
		return exit(err)
	}
	for name, h := range hashes {
		fi, err := os.Stat(name)
		if err != nil {
			return exit(err)
		}
		if _, err := stmt.Exec(name, h.Hash, h.Size, fi.ModTime()); err != nil {
			return exit(err)
		}
	}
	if err := stmt.Close(); err != nil {
		return exit(err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if _, err := db.Exec("VACUUM"); err != nil {
		return err
	}
	return nil
}

func main() {
	// WARN
	// {
	// 	data, err := os.ReadFile("audio_hashes.json")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	var hashes map[string]FileHash
	// 	if err := json.Unmarshal(data, &hashes); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	ctx := context.Background()
	// 	db, err := OpenDatabase(ctx, "audio_hashes.sqlite3")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer db.Close()
	// 	if err := BulkInsertHashes(ctx, db, hashes); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	return
	// }

	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

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
