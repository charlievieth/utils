package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
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
		return fmt.Errorf("invalid type: %[1]T: %#[1]v", src)
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

/*
{
    "session_id": "SESS",
    "ppid": 123,
    "status_code": 1,
    "history_id": 999,
    "command": [
        "cat",
        "bar"
    ]
}
*/
type Request struct {
	PPid       int    `json:"ppid"`
	StatusCode int    `json:"status_code"`
	HistoryID  int    `json:"history_id"` // TODO: do we need this?
	SessionID  string `json:"session_id"` // TODO: use this or the PID?
	Username   string `json:"username"`
	// Time        time.Time `json:"time"`
	Command []string `json:"command"`
}

type DB struct {
	db *sql.DB
}

func NewDB(filename string) (*DB, error) {
	// TODO: tune params
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=rwc", filename))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func (d *DB) Insert(r *Record) error {
	const query = `INSERT INTO history (
	    ppid,
	    status_code,
	    history_id,
	    session_id,
	    server_id,
	    username,
	    created_at,
	    command,
	    arguments,
	    full_command
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`
	_, err := d.db.Exec(query,
		r.PPid,
		r.StatusCode,
		r.HistoryID,
		r.SessionID,
		r.ServerID,
		r.Username,
		r.CreatedAt,
		r.Command,
		r.Arguments,
		r.FullCommand(),
	)
	return err
}

type RequestHandler struct {
	log *zap.Logger
}

func RequestHandler_X(w http.ResponseWriter, r *http.Request) {
	defer func() {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req Request
	if err := dec.Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // WARN
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "reading request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(b, &req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Command) == 0 {

	}

	rec := Record{
		PPid:       req.PPid,
		StatusCode: req.StatusCode,
		HistoryID:  req.HistoryID,
		SessionID:  req.SessionID,
		// ServerID:   req.ServerID,
		Username:  req.Username,
		CreatedAt: time.Now(),
		Command:   req.Command[0],
		Arguments: req.Command[1:],
	}
	_ = rec
}

type Record struct {
	ID         int         `db:"id"`
	PPid       int         `db:"ppid"`
	StatusCode int         `db:"status_code"`
	HistoryID  int         `db:"history_id"` // TODO: do we need this?
	SessionID  string      `db:"session_id"` // TODO: use this or the PID?
	ServerID   string      `db:"server_id"`  // TODO: do we need a per-server ID?
	Username   string      `db:"username"`
	CreatedAt  time.Time   `db:"created_at"`
	Command    string      `db:"command"`   // Command name
	Arguments  StringArray `db:"arguments"` // JSON encoded argv
}

func (r *Record) FullCommand() string {
	switch len(r.Arguments) {
	case 0:
		return r.Command
	case 1:
		return r.Command + " " + r.Arguments[0]
	case 2:
		return r.Command + " " + r.Arguments[0] + " " + r.Arguments[1]
	default:
		n := len(r.Command) + len(r.Arguments) + 1
		for _, s := range r.Arguments {
			n += len(s)
		}
		var b strings.Builder
		b.Grow(n)
		b.WriteString(r.Command)
		for _, s := range r.Arguments {
			b.WriteByte(' ')
			b.WriteString(s)
		}
		return b.String()
	}
}

// TODO: add handlers for initializing and closing new terminal sessions
func OpenSession(w http.ResponseWriter, r *http.Request) {
	panic("TODO")
}

func CloseSession(w http.ResponseWriter, r *http.Request) {
	panic("TODO")
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, fmt.Sprintf("404 page not found: %s", r.URL.Path), http.StatusNotFound)
}

func Reflect(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close() // WARN

	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "    "); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		fmt.Printf("Error: not JSON: %s\n", string(b))
		return
	}
	fmt.Printf("%s\n", &buf)
	w.WriteHeader(200)
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
	w.Write([]byte("okay\n"))
	// WARN
	fmt.Println(r.URL.String())
}

type Config struct {
	UnixSocketAddr string
	LogDir         string
	Log            *zap.Logger
}

func (c *Config) Init() error {
	if c.UnixSocketAddr == "" {
		c.UnixSocketAddr = "${HOME}/.local/share/histdb/socket/sock.sock"
	}
	if c.LogDir == "" {
		c.LogDir = "${HOME}/.local/share/histdb/logs"
	}
	c.UnixSocketAddr = filepath.Clean(os.ExpandEnv(c.UnixSocketAddr))
	c.LogDir = filepath.Clean(os.ExpandEnv(c.LogDir))

	if err := os.MkdirAll(filepath.Dir(c.UnixSocketAddr), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(c.LogDir, 0755); err != nil {
		return err
	}

	zconf := zap.NewProductionConfig()
	zconf.OutputPaths = append(
		zconf.OutputPaths,
		filepath.Join(c.LogDir, "server.log"),
	)
	log, err := zconf.Build(zap.Fields(zap.Int("ppid", os.Getppid())))
	if err != nil {
		return err
	}
	c.Log = log.Named("server")
	return nil
}

type DrainHandler struct {
	h http.Handler
}

func (h *DrainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.h.ServeHTTP(w, r)
	if r != nil && r.Body != nil {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}
}

type Server struct {
	log *zap.Logger
}

type handler struct {
	fn   func(w http.ResponseWriter, r *http.Request) error
	path string
	log  *zap.Logger
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// err :=
}

func CheckActive(addr *net.UnixAddr) error {
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialUnix("unix", nil, addr)
			},
		},
	}
	res, err := client.Get("http://unix/health")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status code (%d): %s", res.StatusCode, res.Status)
	}
	return nil
}

func main() {
	// sock, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	// if err != nil {
	// 	Fatal(err)
	// }
	// sadd := syscall.SockaddrUnix{
	// 	Name: "foo.sock",
	// }
	// if err := syscall.Bind(sock, &sadd); err != nil {
	// 	Fatal(err)
	// }

	conf := Config{
		UnixSocketAddr: "/Users/cvieth/.local/share/histdb/socket/sock.sock",
		LogDir:         "/Users/cvieth/.local/share/histdb/logs",
	}
	if err := conf.Init(); err != nil {
		Fatal(err)
	}
	conf.Log.Info("starting server")

	// TODO: mkdirs and remove old socket

	addr := net.UnixAddr{
		Net:  "unix",
		Name: conf.UnixSocketAddr,
	}

	// Check if a server is currently running
	if err := CheckActive(&addr); err == nil {
		fmt.Println("server already running")
		return
	}

	// Remove the old socket, if any.
	if err := os.Remove(conf.UnixSocketAddr); !os.IsNotExist(err) {
		Fatal(err)
	}

	l, err := net.ListenUnix("unix", &addr)
	if err != nil {
		Fatal(err)
	}
	l.SetUnlinkOnClose(true)
	defer l.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", NotFound)
	mux.HandleFunc("/health", HealthCheck)
	mux.HandleFunc("/reflect", Reflect)

	server := http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second * 30, // probably don't need this
	}
	server.SetKeepAlivesEnabled(false) // WARN

	// NB: listening on unix socket - not TCP
	if err := server.Serve(l); err != nil {
		Fatal(err)
	}
	return

}

/*
	wg := new(sync.WaitGroup)
	go func() {
		for {
			conn, err := l.AcceptUnix()
			if err != nil {
				log.Fatal("error [listen]:", err)
			}
			wg.Add(1)
			go func(conn *net.UnixConn) {
				defer wg.Done()
				defer conn.Close()
				var buf bytes.Buffer
				if _, err := buf.ReadFrom(conn); err != nil {
					Fatal(err)
				}
				fmt.Println(buf.String())
			}(conn)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := net.DialUnix("unix", nil, &addr)
		if err != nil {
			Fatal(err)
		}
		fmt.Fprintln(conn, "Hello!!!")
		defer conn.Close()
	}()

	wg.Wait()
*/

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

/*
func realMain_XXX() error {
	start := time.Now()
	t := start

	if err := parseFlags(); err != nil {
		return fmt.Errorf("error [parse]: %s", err)
	}
	if flag.NArg() == 0 {
		return errors.New("error [usage]: [OPTIONS] [ID] [COMMANDS...]")
	}

	hid, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		return fmt.Errorf("error [id]: %s", err)
	}

	if Debug {
		fmt.Println("parse:", time.Since(t))
		t = time.Now()
	}

	addr := net.UnixAddr{
		Net:  "unix",
		Name: UnixSocketAddr,
	}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		return fmt.Errorf("error [dial]: %s", err)
	}
	defer conn.Close()

	if Debug {
		fmt.Println("conn:", time.Since(t))
		t = time.Now()
	}

	e := Entry{
		Time:       time.Now(),
		PPid:       ParentPid,
		StatusCode: ReturnValue,
		HistoryID:  hid,
		Username:   username(),
		Command:    flag.Args()[1:],
	}
	if err := json.NewEncoder(conn).Encode(&e); err != nil {
		return fmt.Errorf("error [encode]: %s", err)
	}

	if Debug {
		fmt.Println("send:", time.Since(t))
		fmt.Println("Time:", time.Since(start))
	}

	return nil
}
*/
