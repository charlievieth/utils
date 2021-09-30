package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	goflag "flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile)

	// TODO: check if STDOUT is a TTY
	log.SetPrefix("\033[00;33m[WARN]\033[0m ")
}

type TestAction string

const (
	TestRun    = TestAction("run")    // the test has started running
	TestPause  = TestAction("pause")  // the test has been paused
	TestCont   = TestAction("cont")   // the test has continued running
	TestPass   = TestAction("pass")   // the test passed
	TestBench  = TestAction("bench")  // the benchmark printed log output but did not fail
	TestFail   = TestAction("fail")   // the test or benchmark failed
	TestOutput = TestAction("output") // the test printed output
	TestSkip   = TestAction("skip")   // the test was skipped or the package contained no tests
)

type TestEvent struct {
	Time    time.Time  `json:",omitempty"` // encodes as an RFC3339-format string
	Action  TestAction `json:",omitempty"`
	Package string     `json:",omitempty"`
	Test    string     `json:",omitempty"`
	Elapsed float64    `json:",omitempty"` // seconds
	Output  string     `json:",omitempty"`
}

// type TestFunc struct {
// 	Package string `json:",omitempty"`
// 	Name    string `json:",omitempty"`
// }

// func ShouldRecord(events []*TestEvent, pkg, test string) bool {
// 	const RecordMsg = "Do you need to regenerate the recording with the -record flag?"
// 	for _, e := range events {
// 		if e.Package == pkg && e.Test == test {
// 			if strings.Contains(e.Output, RecordMsg) {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }

func ContextCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func ShouldRecord(events []*TestEvent, failed *TestEvent) bool {
	const RecordMsg = "Do you need to regenerate the recording with the -record flag?"
	for _, e := range events {
		if e.Package == failed.Package && e.Test == failed.Test {
			if strings.Contains(e.Output, RecordMsg) {
				return true
			}
		}
	}
	return false
}

func HasSerialTest(events []*TestEvent) bool {
	for _, e := range events {
		if strings.HasPrefix(e.Test, "TestSerial") {
			return true
		}
	}
	return false
}

func RecordTest(ctx context.Context, events []*TestEvent) error {
	if len(events) == 0 {
		return errors.New("no events to re-record")
	}
	pkg, err := build.Import(events[0].Package, ".", build.FindOnly)
	if err != nil {
		return err
	}
	run := eventTestsPattern(events)

	var args []string
	if HasSerialTest(events) {
		args = []string{"test", "-p=1", "-run", run, "-record", pkg.ImportPath}
	} else {
		args = []string{"test", "-run", run, "-record", pkg.ImportPath}
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = pkg.Dir
	out, err := cmd.CombinedOutput()
	os.Stdout.Write(out)
	return err
}

func eventTestsPattern(events []*TestEvent) string {
	if len(events) == 0 {
		return `^$` // run nothing
	}
	tests := make([]string, len(events))
	for i, e := range events {
		s := e.Test
		if i := strings.IndexByte(s, '/'); i != -1 {
			s = s[:i]
		}
		tests[i] = regexp.QuoteMeta(s)
	}
	sort.Strings(tests)
	s := tests[0]
	a := tests[:1]
	for i := 1; i < len(tests); i++ {
		if tests[i] != s {
			a = append(a, tests[i])
			s = tests[i]
		}
	}
	return `^(` + strings.Join(a, "|") + `)$`
}

func iterKeys(m map[string][]*TestEvent) []string {
	a := make([]string, 0, len(m))
	for k := range m {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

func removeIllegalFlags(args []string) []string {
	a := make([]string, 0, len(args))
	for _, s := range args {
		switch s {
		case "-json":
			log.Println("removing extra flag: \"-json\"")
		default:
			a = append(a, s)
		}
	}
	return a
}

func excludeOutput(s string) bool {
	switch s {
	case "PASS\n", "testing: warning: no tests to run\n":
		return true
	}
	return false
}

func runOnce(fn func() error) func() error {
	var once sync.Once
	return func() (err error) {
		once.Do(func() { err = fn() })
		return
	}
}

// TODO:
// 	* match `go test` output (this will be a pain with nested tests)
// 	* add "print longest" option
func realMain() error {
	// TODO: consider renaming to "-out" or something that doesn't
	// conflict with the `go test` flags.
	flag := goflag.NewFlagSet(os.Args[0], goflag.ContinueOnError)
	jsonFile := flag.String("json", "", "write test JSON to FILENAME")
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse(os.Args[1:])

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var bw *bufio.Writer
	var closeWriters = func() error { return nil }
	if *jsonFile != "" {
		f, err := os.OpenFile(*jsonFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		bw = bufio.NewWriter(f)
		closeWriters = runOnce(func() error {
			ef := bw.Flush()
			if ec := f.Close(); ec != nil {
				ef = ec
			}
			return ef
		})
		defer closeWriters()
	}

	args := append([]string{"test", "-json"}, removeIllegalFlags(flag.Args())...)
	cmd := exec.CommandContext(ctx, "go", args...)

	rc, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer rc.Close()
	br := bufio.NewReader(rc)

	// TODO: print only when verbose
	// if *verbose {
	log.Printf("Starting command: %q\n", cmd.Args)
	// }
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command %q: %w", cmd.Args, err)
	}

	var events []*TestEvent
	var readErr error
	for {
		b, e := br.ReadBytes('\n')
		if len(b) != 0 {
			event := &TestEvent{}
			if uerr := json.Unmarshal(b, event); uerr != nil {
				readErr = uerr
				break // this should never happen
			}
			events = append(events, event)
			if event.Action == TestOutput {
				// When verbose == false we only print lines where the Test
				// field is empty. This matches the behavior of `go test`.
				if *verbose || (event.Test == "" && !excludeOutput(event.Output)) {
					if _, werr := os.Stdout.WriteString(event.Output); werr != nil {
						readErr = fmt.Errorf("writing to STDOUT: %w", werr)
						break // no point proceeding
					}
				}
			}
			if bw != nil {
				if _, werr := bw.Write(b); werr != nil {
					log.Println("Error: writing to JSON file:", werr)
					bw.Reset(io.Discard) // drop all future writes
				}
			}
		}
		if e != nil {
			if e != io.EOF {
				readErr = fmt.Errorf("reading cmd output: %w", e)
			}
			break
		}
	}
	if readErr != nil {
		return err
	}

	// TODO: exit non-zero when this occurs
	if err := closeWriters(); err != nil {
		log.Println("Error: closing JSON file:", err)
	}

	cmdErr := cmd.Wait()
	if errors.Is(cmdErr, context.Canceled) {
		log.Println("Context canceled exiting...")
		return cmdErr
	}
	if cmdErr == nil {
		return nil
	}
	fmt.Println("FAIL") // match `go test`

	failed := make(map[string][]*TestEvent)
	nofix := make(map[string][]*TestEvent)
	for _, e := range events {
		if e.Action != TestFail || e.Test == "" {
			continue
		}
		if ShouldRecord(events, e) {
			failed[e.Package] = append(failed[e.Package], e)
		} else {
			nofix[e.Package] = append(nofix[e.Package], e)
		}
	}
	if len(failed) == 0 {
		log.Println("Error: cannot re-record any failed tests")
		// TODO: better error message
		return cmdErr
	}

	// TODO: do this in parallel

	log.Println("Re-Recording tests:")
	var first error
	for _, pkg := range iterKeys(failed) {
		evs := failed[pkg]
		sort.Slice(evs, func(i, j int) bool {
			return evs[i].Test < evs[j].Test
		})
		log.Printf("  %s\n", pkg)
		for _, e := range evs {
			log.Printf("    %s\n", e.Test)
		}

		if err := RecordTest(ctx, evs); err != nil {
			log.Printf("Error recording test %q: %s\n", pkg, err)
			if errors.Is(err, context.Canceled) {
				log.Println("Context canceled exiting...")
				return err
			}
			if first == nil {
				first = err
			}
		}
	}

	if len(nofix) > 0 {
		log.Println("Unable to fix the following tests:")
		for _, pkg := range iterKeys(nofix) {
			evs := nofix[pkg]
			sort.Slice(evs, func(i, j int) bool {
				return evs[i].Test < evs[j].Test
			})
			log.Printf("  %s\n", pkg)
			for _, e := range evs {
				log.Printf("    %s\n", e.Test)
			}
		}
		if first == nil {
			first = errors.New("unfixable test failures")
		}
	}

	return first
}

func main() {
	if err := realMain(); err != nil {
		log.Println("Exiting with error:", err)
		os.Exit(1)
	}
}

func PrintJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		Fatal(err)
	}
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
