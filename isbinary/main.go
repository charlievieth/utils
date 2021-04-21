package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var binaryExtensions = map[string]struct{}{
	".a":               {},
	".adf":             {},
	".atx":             {},
	".bin":             {},
	".bmp":             {},
	".bundle":          {},
	".bz2":             {},
	".cer":             {},
	".cert":            {},
	".class":           {},
	".cot":             {},
	".cpu":             {},
	".dat":             {},
	".db":              {},
	".dbf":             {},
	".der":             {},
	".dylib":           {},
	".elf":             {},
	".eot":             {},
	".exe":             {},
	".expect":          {},
	".expect-noinput":  {},
	".freelist":        {},
	".gdbindexes":      {},
	".gdbtable":        {},
	".gdbtablx":        {},
	".gif":             {},
	".gmo":             {},
	".golden":          {},
	".gox":             {},
	".grb2":            {},
	".gz":              {},
	".h5":              {},
	".hdf":             {},
	".icns":            {},
	".ico":             {},
	".id":              {},
	".img":             {},
	".in":              {},
	".info":            {},
	".its":             {},
	".jar":             {},
	".jp2":             {},
	".jpeg":            {},
	".jpg":             {},
	".macho":           {},
	".map":             {},
	".mo":              {},
	".mp":              {},
	".nc":              {},
	".nib":             {},
	".nit":             {},
	".ntf":             {},
	".o":               {},
	".obj":             {},
	".odp":             {},
	".odt":             {},
	".otf":             {},
	".p12":             {},
	".pbf":             {},
	".pcap":            {},
	".pdf":             {},
	".PNG":             {},
	".png":             {},
	".psd":             {},
	".pyc":             {},
	".raw":             {},
	".rsw":             {},
	".shp":             {},
	".shx":             {},
	".sig":             {},
	".sketch":          {},
	".so":              {},
	".spx":             {},
	".strings":         {},
	".sublime-package": {},
	".syso":            {},
	".tar":             {},
	".tdb":             {},
	".tga":             {},
	".tif":             {},
	".tiff":            {},
	".ttf":             {},
	".UTF8":            {},
	".vic":             {},
	".whl":             {},
	".wkb":             {},
	".woff":            {},
	".woff2":           {},
	".xi":              {},
	".xz":              {},
	".zip":             {},
}

func isBinaryFile(name string, buf []byte) (bool, error) {
	if len(buf) == 0 {
		buf = make([]byte, 512)
	}
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	n, err := f.Read(buf)
	f.Close()
	if err != nil && err != io.EOF {
		return false, err
	}
	return bytes.IndexByte(buf[:n], 0) != -1, nil
}

type File struct {
	Path     string `json:"path"`
	Base     string `json:"base"`
	Ext      string `json:"ext"`
	IsBinary bool   `json:"is_binary"`
}

type byFilePath []File

func (b byFilePath) Len() int           { return len(b) }
func (b byFilePath) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byFilePath) Less(i, j int) bool { return b[i].Path < b[i].Path }

type byIsBinary []File

func (b byIsBinary) Len() int      { return len(b) }
func (b byIsBinary) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func (b byIsBinary) Less(i, j int) bool {
	return b[j].IsBinary && !b[i].IsBinary
}

type Worker struct {
	files     []File
	printOnly bool
}

func (w *Worker) Run(wg *sync.WaitGroup, names <-chan string) {
	defer wg.Done()
	buf := make([]byte, 512)
	for name := range names {
		ext := filepath.Ext(name)
		if ext == "" {
			ext = filepath.Base(name)
		}
		binary, err := isBinaryFile(name, buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %s\n", name, err)
			continue
		}
		if w.printOnly {
			fmt.Printf("%s\t%t\n", ext, binary)
		} else {
			abs, err := filepath.Abs(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s: %s\n", name, err)
				abs = name
			}
			w.files = append(w.files, File{
				Path:     abs,
				Base:     filepath.Base(name),
				Ext:      filepath.Ext(name),
				IsBinary: binary,
			})
		}
	}
}

type Workers []*Worker

func (w Workers) Files() []File {
	n := 0
	for i := range w {
		n += len(w[i].files)
	}
	a := make([]File, 0, n)
	for i := range w {
		a = append(a, w[i].files...)
	}
	return a
}

func XRun(wg *sync.WaitGroup, names <-chan string) {
	defer wg.Done()
	buf := make([]byte, 512)
	for name := range names {
		ext := filepath.Ext(name)
		if ext == "" {
			ext = filepath.Base(name)
		}
		binary, err := isBinaryFile(name, buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %s\n", name, err)
			continue
		}
		fmt.Printf("%s\t%t\n", ext, binary)
	}
}

type ExtBinary struct {
	Ext    string   `json:"ext"`
	Binary []string `json:"binary"`
	Text   []string `json:"text"`
}

func FindIssues(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		Fatal(err)
	}
	var files []File
	if err := json.Unmarshal(data, &files); err != nil {
		Fatal(err)
	}
	m := make(map[string]*ExtBinary)
	for _, f := range files {
		if f.Ext == "" {
			continue
		}
		e := m[f.Ext]
		if e == nil {
			e = &ExtBinary{Ext: f.Ext}
			m[f.Ext] = e
		}
		if f.IsBinary {
			e.Binary = append(e.Binary, f.Path)
		} else {
			e.Text = append(e.Text, f.Path)
		}
	}
	for k, v := range m {
		if len(v.Text) == 0 || len(v.Binary) == 0 {
			delete(m, k)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if err := enc.Encode(m); err != nil {
		Fatal(err)
	}
}

func main() {
	nullTerminated := flag.Bool("0", false, "input is null terminated")
	jsonFlag := flag.Bool("json", false, "input is null terminated")
	dbFlag := flag.String("db", "", "creat a sqlite database of all the files")
	flag.Parse()

	var delim byte = '\n'
	if *nullTerminated {
		delim = 0
	}

	if *dbFlag != "" {
		if _, err := os.Stat(*dbFlag); err == nil {
			Fatal("refusing to overwrite database: " + *dbFlag)
		}
	}

	numCPU := runtime.NumCPU() * 2

	workers := make(Workers, numCPU)
	names := make(chan string, numCPU*2)

	wg := &sync.WaitGroup{}
	for i := 0; i < len(workers); i++ {
		workers[i] = &Worker{
			printOnly: *jsonFlag == false,
		}
		wg.Add(1)
		go workers[i].Run(wg, names)
	}

	var err error
	if flag.NArg() == 0 || flag.Arg(0) == "-" {
		r := bufio.NewReader(os.Stdin)
		for {
			s, e := r.ReadString(delim)
			if len(s) > 0 {
				s = s[:len(s)-1]
				names <- s
			}
			if e != nil {
				if e != io.EOF {
					err = e
				}
				break
			}
		}
	} else {
		for _, name := range flag.Args() {
			names <- name
		}
	}
	close(names)
	if err != nil {
		Fatal(err) // WARN
	}
	wg.Wait()

	var files []File
	if *jsonFlag || *dbFlag != "" {
		files = workers.Files()
		sort.Sort(byIsBinary(files))
		sort.Stable(byFilePath(files))
	}

	if *jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(files); err != nil {
			Fatal(err)
		}
	}
	if *dbFlag != "" {
		if err := CreateDatabase(*dbFlag, files); err != nil {
			Fatal(err)
		}
	}
}

func CreateDatabase(name string, files []File) error {
	const createStmt = `
	CREATE TABLE files (
	    id   INTEGER PRIMARY KEY,
	    path TEXT NOT NULL,
	    base TEXT NOT NULL,
	    ext  TEXT,
	    is_binary BOOL NOT NULL
	);`

	const insertStmt = `INSERT INTO files (
		path,
		base,
		ext,
		is_binary
	) VALUES (?, ?, ?, ?);`

	const baseIndexStmt = `CREATE INDEX ix_files_base ON files(base);`
	const extIndexStmt = `CREATE INDEX ix_files_ext ON files(ext);`
	const binaryIndexStmt = `CREATE INDEX ix_files_binary ON files(is_binary);`

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=rwc", name))
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		Fatal(err) // return err
	}

	if _, err := db.Exec(createStmt); err != nil {
		Fatal(err) // return err
	}

	tx, err := db.Begin()
	if err != nil {
		Fatal(err) // return err
	}
	stmt, err := tx.Prepare(insertStmt)
	if err != nil {
		Fatal(err) // return err
	}
	for _, f := range files {
		var ext interface{}
		if f.Ext != "" {
			ext = f.Ext
		}
		if _, err := stmt.Exec(f.Path, f.Base, ext, f.IsBinary); err != nil {
			tx.Rollback()
			Fatal(err) // return err
		}
	}
	if err := tx.Commit(); err != nil {
		Fatal(err) // return err
	}

	if _, err := db.Exec(baseIndexStmt); err != nil {
		Fatal(err) // return err
	}
	if _, err := db.Exec(extIndexStmt); err != nil {
		Fatal(err) // return err
	}
	if _, err := db.Exec(binaryIndexStmt); err != nil {
		Fatal(err) // return err
	}
	return nil
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

func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	return enc.Encode(v)
}
