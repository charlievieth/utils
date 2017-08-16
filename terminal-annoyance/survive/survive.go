package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var AllFontNames = [...]string{
	"Apple Braille Outline 6 Dot",
	"Apple Braille Outline 8 Dot",
	"Apple Braille Pinpoint 6 Dot",
	"Apple Braille Pinpoint 8 Dot",
	"Apple Braille",
	"Apple Color Emoji",
	"Apple Symbols",
	"AppleSDGothicNeo-Bold",
	"AppleSDGothicNeo-Regular",
	"AquaKana",
	"ArialHB",
	"Avenir Next Condensed",
	"Avenir Next",
	"Avenir",
	"Courier",
	"encodings",
	"fonts",
	"fonts",
	"fonts",
	"GeezaPro",
	"Geneva",
	"HelveLTM",
	"Helvetica LT M",
	"Helvetica",
	"HelveticaNeue",
	"HelveticaNeueDeskInterface",
	"HiraKakuInterface-W1",
	"HiraKakuInterface-W2",
	"Keyboard",
	"Kohinoor",
	"LastResort",
	"LucidaGrande",
	"MarkerFelt",
	"Menlo",
	"Monaco",
	"Noteworthy",
	"Optima",
	"Palatino",
	"STHeiti Light",
	"STHeiti Medium",
	"STHeiti Thin",
	"STHeiti UltraLight",
	"Symbol",
	"Thonburi",
	"Times LT M",
	"Times",
	"TimesLTM",
	"ZapfDingbats",
}

const (
	PidSemaphore = "System.CoreServices.SystemAppearance.Resources.Fonts"
)

var (
	PidDirname  string
	PidFilePath string
	PidFileName string
	PidFile     *os.File
)

func randomPidDirname() string {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		log.Fatal(err)
	}
	name := fmt.Sprintf("com.apple.launchd.%s", base64.StdEncoding.EncodeToString(b)[:1])
	return filepath.Join("/tmp", name)
}

func polluteTempDirectory(n int) {
	for i := 0; i < n; i++ {
		os.MkdirAll(randomPidDirname(), 0755)
	}
}

func isPidDir(dirname string) bool {
	list, err := ioutil.ReadDir(dirname)
	if err != nil {
		return false
	}
	for _, fi := range list {
		if fi.IsDir() && fi.Name() == PidSemaphore {
			return true
		}
	}
	return false
}

func findPidDir() string {
	list, err := ioutil.ReadDir("/tmp")
	if err != nil {
		log.Fatal(err)
	}
	for _, fi := range list {
		name := fi.Name()
		if fi.IsDir() && strings.HasPrefix(name, "com.apple.launchd.") {
			if path := filepath.Join("/tmp", name); isPidDir(path) {
				return path
			}
		}
	}
	return ""
}

func createPidDir() string {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(randomPidDirname(), PidSemaphore)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatal(err)
	}
	return path
}

func init() {
	const mode = os.O_CREATE | os.O_APPEND | os.O_RDWR | os.O_SYNC

	rand.Seed(time.Now().UnixNano())
	if p := findPidDir(); p != "" {
		PidDirname = p
	} else {
		PidDirname = createPidDir()
	}

	// Create pid file
	PidFileName = AllFontNames[rand.Intn(len(AllFontNames))-1]
	PidFilePath = filepath.Join(PidDirname, PidFileName)
	f, err := os.OpenFile(PidFileName, mode, 0666)
	if err != nil {
		log.Fatal(err)
	}
	PidFile = f

	polluteTempDirectory(100)
}

var ProcessNames []string

type byLen []string

func (s byLen) Len() int           { return len(s) }
func (s byLen) Less(i, j int) bool { return len(s[i]) < len(s[j]) }
func (s byLen) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func init() {
	// 0       1   2    3  4    5      6        7    8
	// COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME

	cmd := exec.Command("lsof")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	lines := bytes.Split(stdout.Bytes(), []byte{'\n'})
	for _, line := range lines {
		fields := bytes.Fields(line)
		if len(fields) > 8 {
			ProcessNames = append(ProcessNames, string(fields[8]))
		}
	}

	sort.Sort(byLen(ProcessNames))
}

// func SearchStrings(a []string, x string) int {
// 	return Search(len(a), func(i int) bool { return a[i] >= x })
// }

func continuousBuffer(s []string) bool {
	for i := 0; i < len(s)-1; i++ {
		s0 := s[i+0]
		s1 := s[i+1]
		p0 := *(*uintptr)(unsafe.Pointer(&s0))
		p1 := *(*uintptr)(unsafe.Pointer(&s1))
		if n := p0 + uintptr(len(s0)) + 1; n != p1 {
			return false
		}
	}
	return true
}

func RewriteArgs(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, errors.New("zero length buffer")
	}
	if !continuousBuffer(args) {
		return nil, errors.New("not a continuous buffer")
	}

	start := *(*uintptr)(unsafe.Pointer(&args[0]))
	end := *(*uintptr)(unsafe.Pointer(&args[len(args)-1]))
	length := int(end - start + 1)

	b := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: start,
		Len:  length,
		Cap:  length,
	}))
	return bytes.Replace(b, []byte{0}, []byte{' '}, -1), nil
}

/*
func RewriteArgs() {
	start := *(*uintptr)(unsafe.Pointer(&os.Args[0]))
	end := *(*uintptr)(unsafe.Pointer(&os.Args[len(os.Args)-1]))
	_ = start
	_ = end
	fmt.Println(start)
	fmt.Println(end)
	// fmt.Println(len)
	h := reflect.StringHeader{
		Data: start,
		Len:  int(end - start),
	}
	s := *(*string)(unsafe.Pointer(&h))
	fmt.Println(strings.Replace(s, string(0), " ", -1))
}
*/

func main() {
	XXX()
	return

	b, err := RewriteArgs(os.Args)
	if err != nil {
		Fatal(err)
	}
	fmt.Printf("%q\n", string(b))
}

func XXX() {
	cmd := exec.Command("lsof")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	var s []string

	// 0       1   2    3  4    5      6        7    8
	// COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
	lines := bytes.Split(stdout.Bytes(), []byte{'\n'})
	for _, line := range lines {
		fields := bytes.Fields(line)
		if len(fields) > 8 {
			s = append(s, string(fields[8]))
		}
	}
	sort.Strings(s)
	for i := 0; i < len(s); i++ {
		fmt.Println(s[i])
	}
}

func AddPid(pid int) error {
	f, err := os.OpenFile(PidFilePath, os.O_CREATE|os.O_APPEND|os.O_SYNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(PidFile, "%d\n", pid); err != nil {
		return err
	}
	return PidFile.Sync()
}

func RemovePid(pid int) error {
	bp := append(strconv.AppendInt(make([]byte, 0, 30), int64(pid), 10), '\n')

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(PidFile); err != nil {
		return err
	}
	b := buf.Bytes()
	if !bytes.Contains(b, bp) {
		return nil
	}
	b = bytes.Replace(b, bp, []byte{}, 1)
	return PidFile.Sync()
}

func Fatal(err interface{}) {
	var s string
	if _, file, line, ok := runtime.Caller(1); ok {
		s = fmt.Sprintf("%s:%d", file, line)
	}
	if err != nil {
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
