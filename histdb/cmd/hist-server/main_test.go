package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TempDB(t testing.TB) (*sql.DB, func()) {
	f, err := ioutil.TempFile("", "*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	temp := f.Name()
	// db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=rwc", temp))
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory", temp))
	if err != nil {
		os.Remove(temp)
		t.Fatal(err)
	}
	return db, func() { db.Close(); os.Remove(temp) }
}

func CreateTestTable(t testing.TB, db *sql.DB) {
	const stmt = `
	CREATE TABLE IF NOT EXISTS session_ids (
	    id INTEGER PRIMARY KEY
	);`
	if _, err := db.Exec(stmt); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkInsert(b *testing.B) {
	db, cleanup := TempDB(b)
	b.Cleanup(cleanup)

	CreateTestTable(b, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.Exec(`INSERT INTO session_ids DEFAULT VALUES;`)
		if err != nil {
			b.Fatal(err)
		}
	}

	var rows int64
	err := db.QueryRow(`SELECT COUNT(*) FROM session_ids;`).Scan(&rows)
	if err != nil {
		b.Fatal(err)
	}
	b.Log("Rows:", rows)
}

func BenchmarkPrepared(b *testing.B) {
	db, cleanup := TempDB(b)
	b.Cleanup(cleanup)

	CreateTestTable(b, db)

	stmt, err := db.Prepare(`INSERT INTO session_ids DEFAULT VALUES;`)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stmt.Exec()
		if err != nil {
			b.Fatal(err)
		}
	}

	var rows int64
	err = db.QueryRow(`SELECT COUNT(*) FROM session_ids;`).Scan(&rows)
	if err != nil {
		b.Fatal(err)
	}
	b.Log("Rows:", rows)
}
