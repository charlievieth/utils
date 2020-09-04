package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"
	// "go.uber.org/zap"
)

func SessionID() (uint64, error) {
	var max big.Int
	max.SetInt64(math.MaxInt64)
	n, err := rand.Int(rand.Reader, &max)
	if err != nil {
		return 0, err
	}
	return n.Uint64(), nil
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

func TouchFile(name string) error {
	if fi, err := os.Stat(name); err == nil && fi.Mode().IsRegular() {
		return nil
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte{}); err != nil {
		return err
	}
	return f.Close()
}

/*
func SetupLogger(logFile string) (*zap.Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, err
	}
	if err := TouchFile(logFile); err != nil {
		return nil, err
	}
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{logFile}
	return cfg.Build()
}
*/

// var start = time.Now() // WARN

func main() {

	var r Request

	flag.StringVar(&r.Username, "user", "", "username")
	flag.IntVar(&r.PPid, "ppid", 0, "terminal pid") // WARN: make sure this is correct
	flag.IntVar(&r.StatusCode, "status-code", 0, "status code of the last command")
	flag.StringVar(&r.SessionUUID, "session", "", "session UUID") // WARN: maybe just ID

	flag.Parse()
	if flag.NArg() == 0 {
		return
	}

	const HistDir = "/Users/cvieth/Desktop/xhist"

	if err := os.MkdirAll(HistDir, 0755); err != nil {
		panic(err)
	}
	// log, err := SetupLogger(filepath.Join(HistDir, "client.log"))
	// if err != nil {
	// 	panic(err) // WARN
	// }
	// log.Info("pid", zap.Int("ppid-arg", r.PPid), zap.Int("ppid-cmd", os.Getppid()))

	// WARN
	// log.Warn("args", zap.Strings("args", os.Args[1:]))

	args := flag.Args()
	var err error
	r.HistoryID, err = strconv.Atoi(args[0])
	if err != nil {
		panic(err) // WARN
	}
	r.Command = args[1:]

	f, err := os.OpenFile(filepath.Join(HistDir, "history.json"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err) // WARN
	}

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		panic(err) // WARN
	}
	if err := f.Close(); err != nil {
		panic(err) // WARN
	}

	// log.Info("runtime", zap.Duration("time", time.Since(start)))
	// log.Sync()
}
