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
	"io/fs"
	"log"
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
	"time"

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
	start := time.Now()
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
	log.Printf("%s: %s\n", name, time.Since(start))
	return nil
}

//go:embed sql/create_files_table.sql
var CreateFileTableStmt string

//go:embed sql/create_run_ids_table_stmt.sql
var CreateRunIDsTableStmt string

//go:embed sql/insert_files_statement.sql
var InsertFileStmt string

func createTables(ctx context.Context, db *sql.DB) error {
	for _, stmt := range []string{
		CreateRunIDsTableStmt,
		CreateFileTableStmt,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

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

var supportedHashFuncs = map[string]func() hash.Hash{
	"md5":    md5.New,
	"sha256": sha256.New,
}

func defaultNumWorkers() int {
	numCPU := runtime.NumCPU()
	if numCPU < 4 {
		numCPU = 4
	}
	return numCPU
}

func realMain() error {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: PATHs...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	hashType := flag.String("hash", "md5", "hash function to use (md5 or sha256)")
	hashDB := flag.Bool("db", false, "write hashes to a sqlite database")
	verbose := flag.Bool("verbose", false, "print verbose output")
	hashDBName := flag.String("db-name", "", "name of hash database (implies -db)")
	numWorkers := flag.Int("workers", defaultNumWorkers(), "number of parallel workers to use")

	var globExclude GlobSet
	flag.Var(&globExclude, "exclude", "exclude files GLOB")

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: missing PATH")
		flag.Usage()
		os.Exit(1)
	}

	hashFunc := supportedHashFuncs[*hashType]
	if hashFunc == nil {
		return fmt.Errorf("invalid 'hash' function: %q\n", *hashType)
	}
	if *numWorkers <= 0 {
		return fmt.Errorf("non-positive 'workers' argument: %q\n", *numWorkers)
	}

	if !*verbose {
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(log.Lshortfile)
		log.SetPrefix("[gohashfiles] ")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		<-ctx.Done()
		fmt.Fprintln(os.Stderr, "warn: received signal stopping...")
		stop()
	}()

	var db *sql.DB
	var runID int64
	if *hashDB {
		if *hashDBName == "" {
			*hashDBName = "hashes.sqlite"
		}
		connStr, err := connectionString(*hashDBName)
		if err != nil {
			return err
		}
		db, err = sql.Open("sqlite3", connStr)
		if err != nil {
			return err
		}
		if err := createTables(ctx, db); err != nil {
			return err
		}
		const runIDQuery = `INSERT INTO run_ids DEFAULT VALUES RETURNING id;`
		if err := db.QueryRowContext(ctx, runIDQuery).Scan(&runID); err != nil {
			return err
		}
		if runID == 0 {
			return fmt.Errorf("non-positive run_id: %d", runID)
		}
	}

	wg := new(sync.WaitGroup)
	list := new(FileList)
	workCh := make(chan string, *numWorkers*4)

	for i := 0; i < *numWorkers; i++ {
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

	for _, dir := range flag.Args() {
		log.Println("walking:", dir)
		done := ctx.Done()

		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: walking: %s: %v\n", path, err)
				return nil
			}
			typ := d.Type()
			if typ&fs.ModeSymlink != 0 {
				fi, err := os.Stat(path)
				if err != nil {
					return err
				}
				typ = fi.Mode().Type()
			}
			switch {
			case typ.IsDir():
				if globExclude.Match(path) {
					log.Println("skipping directory:", path)
					return filepath.SkipDir
				}
			case typ.IsRegular():
				if !globExclude.Match(path) {
					select {
					case workCh <- path:
					case <-done:
						return ctx.Err()
					}
				} else {
					log.Println("skipping file:", path)
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: walking path: %s: %v", dir, err)
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
			_, err := stmt.ExecContext(ctx, runID, f.Name, base, pext, f.Hash, f.Size)
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
