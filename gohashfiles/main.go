package main

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"

	_ "github.com/mattn/go-sqlite3"
)

type FileHash struct {
	Name, Hash string
	Size       int64
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
	buf     []byte
	h       hash.Hash
	files   *FileList
	newHash func() hash.Hash
}

func (w *Worker) HashFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if w.buf == nil {
		w.buf = make([]byte, 32*1024)
	}
	if w.h == nil {
		w.h = w.newHash()
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
		Size: fi.Size(),
	})
	return nil
}

const DropFileTableStmt = `DROP TABLE IF EXISTS files;`

//go:embed sql/create_files_table.sql
var CreateFileTableStmt string

//go:embed sql/insert_files_statement.sql
var InsertFileStmt string

func connectionString(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	v := u.Query()
	v.Set("_foreign_keys", "1")
	v.Set("_cache_size", "-4000")
	u.RawQuery = v.Encode()
	return u.String(), nil
}

func dropFilesTable(ctx context.Context, db *sql.DB) error {
	const query = `
	SELECT EXISTS(
		SELECT name FROM sqlite_master WHERE type='table' AND name='files'
	);`
	var exists bool
	if err := db.QueryRowContext(ctx, query).Scan(&exists); err != nil {
		return err
	}
	if exists {
		fmt.Fprintln(os.Stderr, "WARNING: dropping `files` table")
		db.Exec(DropFileTableStmt) // no error check
	}
	return nil
}

var supportedHashFuncs = map[string]func() hash.Hash{
	"md5":    md5.New,
	"sha256": sha256.New,
}

func realMain() error {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: PATHs...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	hashType := flag.String("hash", "md5", "hash function to use (md5 or sha256)")
	hashDB := flag.Bool("db", false, "write hashes to a sqlite database")
	hashDBName := flag.String("db-name", "hashes.sqlite", "name of hash database (implies -db)")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: missing PATH")
		flag.Usage()
		os.Exit(1)
	}

	hashFunc := supportedHashFuncs[*hashType]
	if hashFunc == nil {
		fmt.Fprintln(os.Stderr, "error: invalid 'hash' function: %q", *hashType)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		<-ctx.Done()
		fmt.Fprintln(os.Stderr, "warn: received signal stopping...")
		stop()
	}()

	var db *sql.DB
	if *hashDBName != "" {
		*hashDB = true
	}
	if *hashDB {
		connStr, err := connectionString(*hashDBName)
		if err != nil {
			return err
		}
		db, err = sql.Open("sqlite3", connStr)
		if err != nil {
			return err
		}
		if err := dropFilesTable(ctx, db); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, CreateFileTableStmt); err != nil {
			return err
		}
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
			w := &Worker{
				files:   list,
				newHash: hashFunc,
			}
			for name := range workCh {
				if err := w.HashFile(name); err != nil {
					fmt.Fprintf(os.Stderr, "error: %s: %v\n", name, err)
				}
			}
		}()
	}

	for _, path := range flag.Args() {
		fmt.Println("Walking:", path) // TODO: use log instead
		done := ctx.Done()
		err := filepath.Walk(path, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: walking: %s: %v\n", path, err)
				return nil
			}
			if fi.Mode().IsRegular() {
				select {
				case workCh <- path:
				case <-done:
					return ctx.Err()
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: walking path: %s: %v", path, err)
		}
	}
	close(workCh)
	wg.Wait()

	if db != nil {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		stmt, err := tx.PrepareContext(ctx, InsertFileStmt)
		if err != nil {
			return err
		}
		for _, f := range list.Files() {
			ext := strings.ToLower(filepath.Ext(f.Name))
			base := filepath.Base(f.Name)
			var pext *string
			if ext != "" && ext != base {
				pext = &ext
			}
			_, err := stmt.ExecContext(ctx, f.Name, base, pext, f.Hash, f.Size)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		if err := db.Close(); err != nil {
			return err
		}
	} else {
		// TODO: do we care about / want this?
		m := make(map[string][]string, len(list.Files()))
		for _, h := range list.Files() {
			m[h.Hash] = append(m[h.Hash], h.Name)
		}

		w := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)
		for k, v := range m {
			if len(v) > 1 {
				sort.Strings(v)
			}
			_, err := fmt.Fprintf(w, "%d\t%s\t%s\n", len(v), k, v)
			if err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
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
