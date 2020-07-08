package main

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

func UserCacheDir() (string, error) {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserCacheDir()
}

func UserConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserConfigDir()
}

// TODO: consider using a UUID or something
func SessionID() int64 { return time.Now().UnixNano() }

type Session struct {
	SocketAddr string
	ServerPID  int
}

func (s *Session) WriteFile(name string) error {
	if s.SocketAddr == "" {
		return errors.New("session: SocketAddr is empty")
	}
	if s.ServerPID == 0 {
		return errors.New("session: ServerPID is 0")
	}
	f, err := os.OpenFile(name, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	const format = "export HISTDB_SOCKET_ADDR=%[1]s\n" +
		"export HISTDB_SERVER_PID=%[2]d\n"
	if _, err := fmt.Fprintf(f, format, s.SocketAddr, s.ServerPID); err != nil {
		f.Close()
		os.Remove(name)
	}
	return f.Close()
}

type Request struct {
	PPid        int       `json:"ppid"`
	StatusCode  int       `json:"status_code"`
	HistoryID   int       `json:"history_id"` // TODO: do we need this?
	SessionUUID string    `json:"session_id"` // TODO: use this or the PID?
	Username    string    `json:"username"`
	Time        time.Time `json:"time"`
	Command     []string  `json:"command"`
}

type StringArray []string

func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = (*a)[:0]
		return nil
	}
	switch v := src.(type) {
	case string:
		return json.Unmarshal([]byte(v), a)
	case []byte:
		return json.Unmarshal(v, a)
	default:
		return fmt.Errorf("invalid type: %[1]T -- %#[1]v", src)
	}
}

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

type Record struct {
	ID         int         `db:"id"`
	PPid       int         `db:"ppid"`
	StatusCode int         `db:"status_code"`
	HistoryID  int         `db:"history_id"` // TODO: do we need this?
	SessionID  int64       `db:"session_id"` // TODO: use this or the PID?
	ServerID   string      `db:"server_id"`  // TODO: do we need a per-server ID?
	Username   string      `db:"username"`
	Created_at time.Time   `db:"created_at"`
	Command    string      `db:"command"`   // Command name
	Arguments  StringArray `db:"arguments"` // JSON encoded argv
}

func main() {
	var a StringArray
	if err := a.Scan(`["a"]`); err != nil {
		panic(err)
	}
	fmt.Println(a)
}
