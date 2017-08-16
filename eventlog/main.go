package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Record struct {
	ID             int       `json:"-",db:"id"`
	LogName        string    `json:"LogName",db:"log_name"`
	Category       string    `json:"Category",db:"category"`
	CategoryNumber int       `json:"CategoryNumber",db:"category_number"`
	Data           []byte    `json:"Data,omitempty",db:"data"`
	EntryType      string    `json:"EntryType",db:"entry_type"`
	EventID        int       `json:"EventID",db:"event_id"`
	Index          int       `json:"Index",db:"event_index"`
	InstanceId     int       `json:"InstanceId",db:"instance_id"`
	MachineName    string    `json:"MachineName",db:"machine_name"`
	Message        string    `json:"Message",db:"message"`
	Source         string    `json:"Source",db:"source"`
	TimeGenerated  time.Time `json:"TimeGenerated",db:"time_generated"`
	TimeWritten    time.Time `json:"TimeWritten",db:"time_written"`
	UserName       *string   `json:"UserName,omitempty",db:"user_name"`
}

const CreateTableStmt = `
CREATE TABLE IF NOT EXISTS event_records (
	id              INTEGER PRIMARY KEY,
	log_name        TEXT NOT NULL,
	category        TEXT NOT NULL,
	category_number INTEGER NOT NULL,
	data            BLOB,
	entry_type      TEXT NOT NULL,
	event_id        INTEGER NOT NULL,
	event_index     INTEGER NOT NULL,
	instance_id     INTEGER NOT NULL,
	machine_name    TEXT NOT NULL,
	message         TEXT NOT NULL,
	source          TEXT NOT NULL,
	time_generated  TIMESTAMP NOT NULL,
	time_written    TIMESTAMP NOT NULL,
	user_name       TEXT
);`

const InsertStmt = `
INSERT INTO event_records (
	log_name,
	category,
	category_number,
	data,
	entry_type,
	event_id,
	event_index,
	instance_id,
	machine_name,
	message,
	source,
	time_generated,
	time_written,
	user_name
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`

func InsertRecord(stmt *sql.Stmt, rec *Record) error {
	_, err := stmt.Exec(rec.LogName,
		rec.Category,
		rec.CategoryNumber,
		rec.Data,
		rec.EntryType,
		rec.EventID,
		rec.Index,
		rec.InstanceId,
		rec.MachineName,
		rec.Message,
		rec.Source,
		rec.TimeGenerated,
		rec.TimeWritten,
		rec.UserName,
	)
	return err
}

func Usage() {
	const format = `
Usage: %s EVENT_LOG DATABASE_NAME

  Imports JSON EventLog records EVENT_LOG into Sqlite3 DATABASE_NAME.

`
	fmt.Fprintf(os.Stderr, format, filepath.Base(os.Args[0]))
	os.Exit(2)
}

func main() {
	if len(os.Args) != 3 {
		Usage()
	}
	dbname := os.Args[1]
	recfile := os.Args[2]

	f, err := os.Open(recfile)
	if err != nil {
		Fatal(err, 1)
	}
	defer f.Close()

	if fi, err := os.Stat(dbname); err == nil {
		if fi.IsDir() {
			Fatal(fmt.Sprintf("cannot use directory (%s) as a sqlite database", dbname), 1)
		}
		Fatal(fmt.Sprintf("refusing to overwrite database file: %s", dbname), 1)
	}
	db, err := sql.Open("sqlite3", dbname)
	if err != nil {
		Fatal(err, 1)
	}
	defer db.Close()

	if _, err := db.Exec(CreateTableStmt); err != nil {
		Fatal(err, 1)
	}

	// insert records in a transaction
	tx, err := db.Begin()
	if err != nil {
		Fatal(err, 1)
	}
	// abort will rollback the transaction and delete the db file.
	abort := func(err interface{}) {
		os.Remove(dbname)
		tx.Rollback()
		Fatal(err, 2) // skip 2 stack frames
	}

	// use a prepared statement to speed insertion
	stmt, err := tx.Prepare(InsertStmt)
	if err != nil {
		abort(err)
	}

	dec := json.NewDecoder(f)
	for n := 0; err == nil; n++ {
		var rec Record
		if err = dec.Decode(&rec); err != nil && err != io.EOF {
			abort(fmt.Sprintf("decoding record (%d): %s", n, err))
		}
		if err := InsertRecord(stmt, &rec); err != nil {
			abort(fmt.Sprintf("inserting record (%d): %s", n, err))
		}
	}
	if err := stmt.Close(); err != nil {
		abort(err)
	}
	if err := tx.Commit(); err != nil {
		abort(err)
	}
}

func Fatal(err interface{}, skip int) {
	if err != nil {
		if skip < 1 {
			skip = 1
		}
		var format string
		if _, file, line, ok := runtime.Caller(skip); ok && file != "" {
			format = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
		} else {
			format = "Error"
		}
		switch err.(type) {
		case error, string, fmt.Stringer:
			fmt.Fprintf(os.Stderr, "%s: %s\n", format, err)
		default:
			fmt.Fprintf(os.Stderr, "%s: %#v\n", format, err)
		}
		os.Exit(1)
	}
}
