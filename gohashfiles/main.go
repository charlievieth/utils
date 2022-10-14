package main

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
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

var debug = log.New(io.Discard, "[debug] ", log.Lshortfile)

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("[error] ")
}

type FileHash struct {
	Name, Hash string
	Size       int64
	RunID      int64
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

	ctx   context.Context
	runID int64
	stmt  *sql.Stmt
	json  *jsonFileEncoder
}

func (w *Worker) doWork(wg *sync.WaitGroup, ch <-chan string) {
	defer wg.Done()
	done := w.ctx.Done()
	n := 0
	for name := range ch {
		if err := w.HashFile(name); err != nil {
			log.Printf("%s: %v\n", name, err)
		}
		// check of the channel is closed
		if n != 0 && n%128 == 0 {
			select {
			case <-done:
				return
			default:
			}
		}
	}
}

// type BatchInerster struct {
// 	mu    sync.Mutex
// 	db    *sql.DB
// 	stmt  *sql.Stmt
// 	tx    *sql.Tx
// 	count int64
// }

// func (b *BatchInerster) Insert(f *FileHash) error {
// 	ext := strings.ToLower(filepath.Ext(f.Name))
// 	base := filepath.Base(f.Name)
// 	var pext *string
// 	if ext != "" && ext != base {
// 		pext = &ext
// 	}
// 	b.mu.Lock()
// 	defer b.mu.Unlock()
// 	_, err := b.stmt.Exec(f.RunID, f.Name, base, pext, f.Hash, f.Size)
// 	if err != nil {
// 		return err
// 	}
// 	b.count++
// 	if b.count == 128 {
//
// 	}
// 	return err
// }

func (w *Worker) InsertFile(f *FileHash) error {
	ext := strings.ToLower(filepath.Ext(f.Name))
	base := filepath.Base(f.Name)
	var pext *string
	if ext != "" && ext != base {
		pext = &ext
	}
	_, err := w.stmt.Exec(f.RunID, f.Name, base, pext, f.Hash, f.Size)
	return err
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
	hashd := time.Since(start)
	file := FileHash{
		Name:  name,
		Hash:  hex.EncodeToString(w.h.Sum(nil)),
		Size:  fi.Size(),
		RunID: w.runID,
	}
	// TODO: batch these
	var err1 error
	var err2 error
	if w.stmt != nil {
		err1 = w.InsertFile(&file)
	}
	if w.json != nil {
		err2 = w.json.Encode(&file)
	}
	w.files.Add(file)
	totald := time.Since(start)
	debug.Printf("%s: hash: %s total: %s\n", name, hashd, totald)
	if err1 != nil {
		return err1
	}
	return err2
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
	v.Set("_mutex", "full")
	v.Set("_journal_mode", "TRUNCATE")
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

func ignoreError(err error) bool {
	return err == nil || os.IsPermission(err) || os.IsNotExist(err)
}

type jsonFileEncoder struct {
	mu     sync.Mutex
	file   *os.File
	closed bool
}

func newJSONFileEncoder(name string) (*jsonFileEncoder, error) {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &jsonFileEncoder{file: f}, nil
}

var newLine = []byte{'\n'}

func (e *jsonFileEncoder) Encode(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, err = e.file.Write(data); err == nil {
		_, err = e.file.Write(newLine)
	}
	return err
}

func (e *jsonFileEncoder) Close() error {
	if e == nil {
		return os.ErrInvalid
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.file == nil {
		return os.ErrInvalid
	}
	if e.closed {
		return os.ErrClosed
	}
	e.closed = true
	return e.file.Close()
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
	jsonFile := flag.String("json", "", "name of file to write JSON to")
	numWorkers := flag.Int("workers", defaultNumWorkers(), "number of parallel workers to use")

	_ = jsonFile

	var globExclude GlobSet
	flag.Var(&globExclude, "exclude", "exclude files GLOB")

	flag.Parse()
	if flag.NArg() == 0 {
		return fmt.Errorf("missing PATH argument")
	}

	hashFunc := supportedHashFuncs[*hashType]
	if hashFunc == nil {
		return fmt.Errorf("invalid 'hash' function: %q\n", *hashType)
	}
	if *numWorkers <= 0 {
		return fmt.Errorf("non-positive 'workers' argument: %q\n", *numWorkers)
	}
	if *verbose {
		debug.SetOutput(os.Stderr)
	}

	var encoder *jsonFileEncoder
	if *jsonFile != "" {
		var err error
		encoder, err = newJSONFileEncoder(*jsonFile)
		if err != nil {
			return err
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		<-ctx.Done()
		log.Println("warn: received signal stopping...")
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
		defer db.Close()
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

	var stmt *sql.Stmt
	var tx *sql.Tx
	if db != nil {
		var err error
		if tx, err = db.Begin(); err != nil {
			return err
		}
		if stmt, err = tx.Prepare(InsertFileStmt); err != nil {
			return err
		}
		defer func() {
			stmt.Close()
			tx.Commit()
		}()
	}

	wg := new(sync.WaitGroup)
	list := new(FileList)
	workCh := make(chan string, *numWorkers*4)

	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		w := &Worker{
			files:   list,
			newHash: hashFunc,
			runID:   runID,
			stmt:    stmt,
			ctx:     ctx,
			json:    encoder,
		}
		go w.doWork(wg, workCh)
	}

	for _, dir := range flag.Args() {
		dir = filepath.Clean(dir)
		debug.Println("walking:", dir)

		done := ctx.Done()
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Printf("walking: %s: %v\n", path, err)
				if !ignoreError(err) {
					return err
				}
				return nil
			}
			typ := d.Type()
			if typ&fs.ModeSymlink != 0 {
				fi, err := os.Stat(path)
				if err != nil {
					log.Printf("walking: %s: %v\n", path, err)
					if !ignoreError(err) {
						return err
					}
					return nil
				}
				typ = fi.Mode().Type()
			}
			switch {
			case typ.IsDir():
				if globExclude.Exclude(path) {
					debug.Println("skipping directory:", path)
					return filepath.SkipDir
				}
			case typ.IsRegular():
				if !globExclude.Exclude(path) {
					select {
					case workCh <- path:
					case <-done:
						return ctx.Err()
					}
				} else {
					debug.Println("skipping file:", path)
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("walking path: %s: %v\n", dir, err)
		}
	}
	close(workCh)
	wg.Wait()

	if err := tx.Commit(); err != nil {
		log.Println("commiting transaction:", err)
	}
	if err := db.Close(); err != nil {
		log.Println("closing database connection:", err)
	}

	if db != nil {
		// tx, err := db.BeginTx(ctx, nil)
		// if err != nil {
		// 	return err
		// }
		// stmt, err := tx.PrepareContext(ctx, InsertFileStmt)
		// if err != nil {
		// 	return err
		// }
		// for _, f := range list.Files() {
		// 	ext := strings.ToLower(filepath.Ext(f.Name))
		// 	base := filepath.Base(f.Name)
		// 	var pext *string
		// 	if ext != "" && ext != base {
		// 		pext = &ext
		// 	}
		// 	_, err := stmt.ExecContext(ctx, runID, f.Name, base, pext, f.Hash, f.Size)
		// 	if err != nil {
		// 		tx.Rollback()
		// 		return err
		// 	}
		// }
		// if err := tx.Commit(); err != nil {
		// 	return err
		// }
		// if err := db.Close(); err != nil {
		// 	return err
		// }
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
var bytesRead uint64

func copyFile(dst io.Writer, src *os.File, buf []byte) (written int64, err error) {
	const interval = time.Second / 10
	if buf == nil {
		buf = make([]byte, 32*1024)
	}

	timer := globalTimerPool.Get(interval)
	defer globalTimerPool.Put(timer)
	tick := make(chan struct{})
	defer close(tick)

	go func() {
		for {
			select {
			case _, ok := <-tick:
				if !ok {
					return
				}
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(interval)
			case <-timer.C:

			}
		}
	}()

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			select {
			case tick <- struct{}{}:
			default:
			}
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return 0, nil
}

// global pool of *time.Timer's. can be used by multiple goroutines concurrently.
var globalTimerPool timerPool

// timerPool provides GC-able pooling of *time.Timer's.
// can be used by multiple goroutines concurrently.
type timerPool struct {
	p sync.Pool
}

// Get returns a timer that completes after the given duration.
func (tp *timerPool) Get(d time.Duration) *time.Timer {
	if t, _ := tp.p.Get().(*time.Timer); t != nil {
		t.Reset(d)
		return t
	}

	return time.NewTimer(d)
}

// Put pools the given timer.
//
// There is no need to call t.Stop() before calling Put.
//
// Put will try to stop the timer before pooling. If the
// given timer already expired, Put will read the unreceived
// value if there is one.
func (tp *timerPool) Put(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}

	tp.p.Put(t)
}
*/
