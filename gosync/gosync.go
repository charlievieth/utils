package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Watchman Types: https://facebook.github.io/watchman/docs/expr/type.html
//
// b: block special file
// c: character special file
// d: directory
// f: regular file
// p: named pipe (fifo)
// l: symbolic link
// s: socket
// D: Solaris Door
type FileType os.FileMode

func (f FileType) MarshalJSON() ([]byte, error) {
	m := os.FileMode(f)
	if m&os.ModeDir != 0 {
		return []byte{'d'}, nil
	}
	if m.IsRegular() {
		return []byte{'f'}, nil
	}
	if m&os.ModeDevice != 0 {
		return []byte{'b'}, nil
	}
	if m&os.ModeDevice|os.ModeCharDevice != 0 {
		return []byte{'c'}, nil
	}
	if m&os.ModeNamedPipe != 0 {
		return []byte{'p'}, nil
	}
	if m&os.ModeSymlink != 0 {
		return []byte{'l'}, nil
	}
	if m&os.ModeSocket != 0 {
		return []byte{'s'}, nil
	}
	if m&os.ModeIrregular != 0 {
		return []byte{'D'}, nil
	}
	return nil, errors.New("invalid FileMode: " + m.String())
}

func (f *FileType) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) != 1 {
		return errors.New("invalid FileMode: " + string(data))
	}
	var m os.FileMode
	switch data[0] {
	case 'b':
		m = os.ModeDevice // TODO: is this correct?
	case 'c':
		m = os.ModeDevice | os.ModeCharDevice // TODO: is this correct?
	case 'd':
		m = os.ModeDir
	case 'f':
		m = 0 // regular file
	case 'p':
		m = os.ModeNamedPipe
	case 'l':
		m = os.ModeSymlink
	case 's':
		m = os.ModeSocket
	case 'D':
		m = os.ModeIrregular // D: Solaris Door
	default:
		return errors.New("invalid FileMode: " + string(data))
	}
	*f = FileType(m)
	m.IsRegular()
	return nil
}

type UnixTime int64

func (u *UnixTime) Time() time.Time {
	if u != nil {
		return time.Unix(0, int64(*u))
	}
	return time.Time{}
}

func (u UnixTime) String() string { return u.Time().String() }

// Watchman file JSON: https://facebook.github.io/watchman/docs/cmd/query.html
//
// name - string: the filename, relative to the watched root
// exists - bool: true if the file exists, false if it has been deleted
// cclock - string: the “created clock”; the clock value when we first observed the file, or the clock value when it last switched from !exists to exists.
// oclock - string: the “observed clock”; the clock value where we last observed some change in this file or its metadata.
// ctime, ctime_ms, ctime_us, ctime_ns, ctime_f - last inode change time measured in integer seconds, milliseconds, microseconds, nanoseconds or floating point seconds respectively.
// mtime, mtime_ms, mtime_us, mtime_ns, mtime_f - modified time measured in integer seconds, milliseconds, microseconds, nanoseconds or floating point seconds respectively.
// size - integer: file size in bytes
// mode - integer: file (or directory) mode expressed as a decimal integer
// uid - integer: the owning uid
// gid - integer: the owning gid
// ino - integer: the inode number
// dev - integer: the device number
// nlink - integer: number of hard links
// new - bool: whether this entry is newer than the since generator criteria
// type - string: the file type. Has the the values listed in the type query expression
// symlink_target - string: the target of a symbolic link if the file is a symbolic link
// content.sha1hex - string: the SHA-1 digest of the file’s byte content, encoded as 40 hexidecimal digits (e.g. "da39a3ee5e6b4b0d3255bfef95601890afd80709" for an empty file)
//
type WatchmanFile struct {
	Name          string   `json:"name"`
	Exists        bool     `json:"exists"`
	CClock        string   `json:"cclock"`
	OClock        string   `json:"oclock"`
	Size          int64    `json:"size"`
	CTime         UnixTime `json:"ctime_ns"`
	MTime         UnixTime `json:"mtime_ns"`
	Mode          int      `json:"mode"`
	Uid           int      `json:"uid"`
	Gid           int      `json:"gid"`
	Ino           int      `json:"ino"`
	Dev           int      `json:"dev"`
	NLink         int      `json:"nlink"`
	New           bool     `json:"new"`
	Type          FileType `json:"type"`
	SymlinkTarget *string  `json:"symlink_target"`
	SHA1          *string  `json:"content.sha1hex"`
}

// ["name", "exists", "cclock", "oclock", "size", "ctime_ns", "mtime_ns", "mode", "uid", "gid", "ino", "dev", "nlink", "new", "type", "symlink_target"]

type SubscriptionResponse struct {
	Files           []File `json:"files"`
	Root            string `json:"root"`
	Subscription    string `json:"subscription"`
	Clock           string `json:"clock"`
	Since           string `json:"since"`
	Version         string `json:"version"`
	IsFreshInstance bool   `json:"is_fresh_instance"`
	Unilateral      bool   `json:"unilateral"`
}

type Action uint8

const (
	DefaultAction Action = iota
	DeleteAction
	UpdateAction
	CreateAction
	RenameAction
)

type Request struct {
	Action  Action
	Payload json.RawMessage
}

func (r *Request) DeleteRequest() (*DeleteRequest, error) {
	v := &DeleteRequest{}
	if err := json.Unmarshal(r.Payload, v); err != nil {
		return nil, fmt.Errorf("unmarshal DeleteRequest: %w", err)
	}
	return v, nil
}

func (r *Request) RenameRequest() (*RenameRequest, error) {
	v := &RenameRequest{}
	if err := json.Unmarshal(r.Payload, v); err != nil {
		return nil, fmt.Errorf("unmarshal RenameRequest: %w", err)
	}
	return v, nil
}

func (r *Request) CreateRequest() (*CreateRequest, error) {
	v := &CreateRequest{}
	if err := json.Unmarshal(r.Payload, v); err != nil {
		return nil, fmt.Errorf("unmarshal CreateRequest: %w", err)
	}
	return v, nil
}

func (r *Request) UpdateRequest() (*UpdateRequest, error) {
	v := &UpdateRequest{}
	if err := json.Unmarshal(r.Payload, v); err != nil {
		return nil, fmt.Errorf("unmarshal UpdateRequest: %w", err)
	}
	return v, nil
}

type DeleteRequest struct {
	Path string `json:"path"`
}

type RenameRequest struct {
	From string      `json:"from"`
	To   string      `json:"to"`
	Mode os.FileMode `json:"mode"`
}

type CreateRequest struct {
	Path string      `json:"path"`
	Data []byte      `json:"data"`
	Mode os.FileMode `json:"mode"`
}

type UpdateRequest struct {
	Action Action
	Path   string
	Data   []byte
	Mode   os.FileMode
}

func WriteFile(name string, data []byte, perm os.FileMode) error {
	dir, base := filepath.Split(name)

	f, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Chmod(perm); err != nil {
		return err
	}
	oldname := f.Name()
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(oldname, name); err != nil {
		os.Remove(oldname)
		return err
	}
	return nil
}

func (r *UpdateRequest) Run() error {
	// TODO: need to calculate relative path

	return nil
}

func isDir(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsDir()
}

func WatchmanHandler(w http.ResponseWriter, r *http.Request) {
	var wq WatchmanFile
	_ = wq
}

func Handler(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}
	io.Copy(io.Discard, r.Body) // consume body
	switch req.Action {
	case DeleteAction:
	case UpdateAction:
	case CreateAction:
	case RenameAction:
	default:
		// ERROR
		return
	}
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}
	switch req.Action {
	case DefaultAction:
		// Error
	case DeleteAction:
		if isDir(req.Path) {
			if err := os.RemoveAll(req.Path); err != nil {
				return
			}
		} else {
			if err := os.Remove(req.Path); err != nil {
				return
			}
		}
	case UpdateAction:
	case CreateAction:
		if req.Mode.IsDir() {
			if err := os.MkdirAll(req.Path, req.Mode); err != nil {
				return
			}
		}
	}
}

func main() {

}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
	}
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

type Filter struct {
	mu    sync.Mutex
	files map[string]string // path => sha1hex
}

func (f *Filter) DidChange(file *File) bool {
	if file.GetSHA1() == "" {
		return true
	}
	f.mu.Lock()
	changed := f.files[file.GetName()] != file.GetSHA1()
	if changed {
		if f.files == nil {
			f.files = make(map[string]string)
		}
		f.files[file.GetName()] = file.GetSHA1()
	}
	f.mu.Unlock()
	return changed
}

type File struct {
	Name          *string   `json:"name,omitempty"`
	Exists        *bool     `json:"exists,omitempty"`
	CClock        *string   `json:"cclock,omitempty"`
	OClock        *string   `json:"oclock,omitempty"`
	Size          *int64    `json:"size,omitempty"`
	CTime         *UnixTime `json:"ctime_ns,omitempty"`
	MTime         *UnixTime `json:"mtime_ns,omitempty"`
	Mode          *int      `json:"mode,omitempty"`
	Uid           *int      `json:"uid,omitempty"`
	Gid           *int      `json:"gid,omitempty"`
	Ino           *int      `json:"ino,omitempty"`
	Dev           *int      `json:"dev,omitempty"`
	NLink         *int      `json:"nlink,omitempty"`
	New           *bool     `json:"new,omitempty"`
	Type          *FileType `json:"type,omitempty"`
	SymlinkTarget *string   `json:"symlink_target,omitempty"`
	SHA1          *string   `json:"content.sha1hex,omitempty"`
}

func (f *File) GetName() string {
	if f != nil && f.Name != nil {
		return *f.Name
	}
	return ""
}

func (f *File) GetExists() bool {
	if f != nil && f.Exists != nil {
		return *f.Exists
	}
	return false
}

func (f *File) GetCClock() string {
	if f != nil && f.CClock != nil {
		return *f.CClock
	}
	return ""
}

func (f *File) GetOClock() string {
	if f != nil && f.OClock != nil {
		return *f.OClock
	}
	return ""
}

func (f *File) GetSize() int64 {
	if f != nil && f.Size != nil {
		return *f.Size
	}
	return 0
}

func (f *File) GetCTime() UnixTime {
	if f != nil && f.CTime != nil {
		return *f.CTime
	}
	return 0
}

func (f *File) GetMTime() UnixTime {
	if f != nil && f.MTime != nil {
		return *f.MTime
	}
	return 0
}

func (f *File) GetMode() int {
	if f != nil && f.Mode != nil {
		return *f.Mode
	}
	return 0
}

func (f *File) GetUid() int {
	if f != nil && f.Uid != nil {
		return *f.Uid
	}
	return 0
}

func (f *File) GetGid() int {
	if f != nil && f.Gid != nil {
		return *f.Gid
	}
	return 0
}

func (f *File) GetIno() int {
	if f != nil && f.Ino != nil {
		return *f.Ino
	}
	return 0
}

func (f *File) GetDev() int {
	if f != nil && f.Dev != nil {
		return *f.Dev
	}
	return 0
}

func (f *File) GetNLink() int {
	if f != nil && f.NLink != nil {
		return *f.NLink
	}
	return 0
}

func (f *File) GetNew() bool {
	if f != nil && f.New != nil {
		return *f.New
	}
	return false
}

func (f *File) GetType() FileType {
	if f != nil && f.Type != nil {
		return *f.Type
	}
	return 0
}

func (f *File) GetSymlinkTarget() string {
	if f != nil && f.SymlinkTarget != nil {
		return *f.SymlinkTarget
	}
	return ""
}

func (f *File) GetSHA1() string {
	if f != nil && f.SHA1 != nil {
		return *f.SHA1
	}
	return ""
}
