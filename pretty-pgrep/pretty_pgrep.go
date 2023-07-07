package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v3/process"
)

var _ = process.Running
var _ = color.NoColor

// func init() {
// 	log.SetOutput(os.Stdout)
// 	log.SetFlags(log.Lshortfile)
// }

type Arg struct {
	Name, Value string
}

type Command struct {
	PID  int32
	Name string
	Args []Arg
}

func containsPathSep(s string) bool {
	if os.PathSeparator == '/' {
		return strings.Contains(s, "/")
	}
	return strings.ContainsAny(s, "/"+string(os.PathSeparator))
}

func ParseCommandName(cmd string) (string, string, error) {
	if !strings.Contains(cmd, " ") {
		if _, err := exec.LookPath(cmd); err != nil {
			return "", "", err
		}
		return cmd, "", nil
	}
	first, rest, ok := strings.Cut(cmd, " ")
	if ok && !containsPathSep(first) {
		return first, rest, nil
	}
	// fmt.Println("HIT:", cmd)
	a := strings.Split(cmd, " ")
	for i := len(a) - 1; i >= 0; i-- {
		p := strings.Join(a[:i+1], " ")
		// fmt.Printf("  P: %q\n", p)
		if _, err := os.Stat(p); err == nil {
			return p, strings.Join(a[i+1:], " "), nil
		}
	}
	return "", "", nil
}

func ParseCommand(line string) (*Command, error) {
	ps, s, ok := strings.Cut(line, " ")
	if !ok {
		return nil, fmt.Errorf("invalid line: %q", line)
	}
	pid, err := strconv.ParseInt(ps, 10, 32)
	if err != nil {
		return nil, err
	}
	_ = s
	_ = pid

	return nil, nil
}

// WARN: remove if not used
type fatalError error

var colorCache map[color.Attribute]*color.Color

// Get cached color
func cc(a color.Attribute) *color.Color {
	if c := colorCache[a]; c != nil {
		return c
	}
	c := color.New(a)
	if colorCache == nil {
		colorCache = make(map[color.Attribute]*color.Color)
	}
	colorCache[a] = c
	return c
}

func PrintProcess(p *Process) error {
	name := p.Name
	if name == "" {
		name = "<none>"
	}
	cc(color.FgHiBlack).Printf("%d: ", p.Proc.Pid)
	cc(color.FgGreen).Printf("%s\n", p.Name)

	// TODO: join args
	for i := 1; i < len(p.Args); i++ {
		cc(color.FgHiBlack).Printf("    %s\n", p.Args[i])
	}
	return nil
}

// WARN: see below
// TODO: use this so that we don't have to request data twice
type Process struct {
	Proc *process.Process
	Name string
	Args []string
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s: [flags...] pattern...\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	// TODO: add option to print full path (first arg)
	nocase := flag.Bool("i", false, "Case-insensitive match.")
	full := flag.Bool("f", false, "Match against full argument lists. "+
		"The default is to match against process names.")
	verbose := flag.Bool("v", false, "Verbose output.")
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	log.SetPrefix("# ")
	log.SetFlags(log.Lshortfile)
	log.SetOutput(io.Discard)
	if *verbose {
		log.SetOutput(os.Stderr)
	}

	args := flag.Args()
	for i, p := range args {
		if _, err := regexp.Compile(p); err != nil {
			log.Fatalf("error: failed to compile pattern `%s`: %s", p, err)
		}
		args[i] = "(" + p + ")"
	}
	pattern := strings.Join(args, "|")
	if *nocase {
		pattern = "(?i)(" + pattern + ")"
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatalf("fatal: parsing combined regexp pattern `%s`: %s", pattern, err)
	}
	if *verbose {
		log.Printf("regexp: `%s`", re.String())
	}

	procs, err := process.Processes()
	if err != nil {
		log.Fatal(err)
	}
	var all []Process
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			log.Printf("error: %d: failed to get proc name: %s\n", p.Pid, err)
			continue
		}
		cmd, err := p.CmdlineSlice()
		if err != nil && *verbose {
			log.Printf("error: %d: failed to get command line: %s\n", p.Pid, err)
		}
		if *full && len(cmd) > 0 {
			name += " " + strings.Join(cmd[1:], " ")
		}
		if re.MatchString(name) {
			all = append(all, Process{p, name, cmd})
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Proc.Pid < all[j].Proc.Pid
	})
	for _, p := range all {
		// name, err := p.Name()
		// if err != nil {
		// 	log.Printf("error: %d: failed to get proc name: %s\n", p.Pid, err)
		// 	continue
		// }
		// cmd, err := p.Cmdline()
		// if err != nil {
		// 	log.Printf("error: %d: failed to get proc name: %s\n", p.Pid, err)
		// 	continue
		// }
		PrintProcess(&p)
		// fmt.Printf("%d %s %s\n", p.Proc.Pid, p.Name, strings.Join(p.Args, " "))
	}
}
