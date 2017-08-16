package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "%s: [utility [argument ...]]\n", filepath.Base(os.Args[0]))
	os.Exit(1)
}

func LookPath(file string) (string, error) {
	if filepath.Base(file) != file {
		return file, nil
	}
	lp, err := exec.LookPath(file)
	if err != nil {
		return "", err
	}
	return lp, nil
}

// Environ, strips program specific variables from the environment.
func Environ() []string {
	const prefix = "__PIPE_" // placeholder for program specific env variables
	e := os.Environ()
	for i, n := 0, 0; i < len(e); i++ {
		if !strings.HasPrefix(e[i], prefix) {
			e[n] = e[i]
			n++
		}
	}
	return e
}

// ExitCode, returns the exit code for command cmd.
//
// TODO: document or standardize exit codes
func ExitCode(cmd *exec.Cmd) (int, error) {
	if cmd.ProcessState == nil {
		return 3, errors.New("exit code: nil process state")
	}
	switch v := cmd.ProcessState.Sys().(type) {
	case syscall.WaitStatus:
		return v.ExitStatus(), nil
	default:
		return 4, fmt.Errorf("exit code: unsuported type: %T", v)
	}
}

func FindProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err == nil {
		p.Release() // Close process handle
	}
	return err
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	// Start collecting all signals immediately.  Any buffered signals will be
	// sent to the command once it starts.
	//
	// We want to be an invisible layer between the service wrapper and the
	// command - so we should not dictate how signals are handled, but instead
	// rely upon the commands signal handling - if it exits, we exit - if it
	// ignores the signal, we ignore it.
	sigCh := make(chan os.Signal, 64)
	signal.Notify(sigCh)

	// NB: We could remove LookPath as any error there is returned on Start().
	name := os.Args[1]
	path, err := LookPath(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid command name (%s): %s\n", name, err)
		os.Exit(1)
	}
	args := os.Args[2:]

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = Environ()

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "starting command (%s): %s\n", name, err)
		os.Exit(1)
	}

	// Closed when Wait() returns.
	haltCh := make(chan struct{})

	// Immediately start waiting for the command to exit.
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait(); close(haltCh) }()

	// Critical section: make sure we do not miss a fast exiting process.
	//
	// NB: This may not be necessary and in fact the errors we've seen
	// running packaging scripts may be due to how we're handling them.
	//
	// I ran this program against a C program 'main () { return 123; }'
	// and Go caught the exit code.  On Windows, that C program compiles
	// down to a single instruction 'mov eax, 123' so it's hard to imagine
	// anything returning faster.
	//
	critCh := make(chan error, 1)
	go func(pid int) {
		done := time.After(time.Second * 10)
		tick := time.NewTicker(time.Millisecond * 100)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := FindProcess(pid); err != nil {
					critCh <- err
					return
				}
			case <-done:
				return
			case <-haltCh:
				return
			}
		}
	}(cmd.Process.Pid)

	// Send any buffered and all future Signals to command.
	go func() {
		for {
			select {
			case sig := <-sigCh:
				// NB: Error handling is for debugging only.  Errors here
				// will most likely result from the command having already
				// exited and in that case we're already on our way out.
				//
				// TODO: make sure SIGINT (really, CTRL-C is being sent on
				// Windows, this is done in winsw by attaching a console to
				// the process).
				if err := cmd.Process.Signal(sig); err != nil {
					fmt.Fprintf(os.Stderr, "sending signal (%s): %s\n", sig, err)
				}
			case <-haltCh:
				return
			}
		}
	}()

	// Wait for termination.
	select {
	case err := <-waitCh:
		fmt.Fprintf(os.Stderr, "wait: %v\n", err)
	case err := <-critCh:
		// If waitCh and critCh are sent simultaneously the selected channel
		// is random - give preference to waitCh as it represents the process
		// ending - not just an error finding it.
		select {
		case e := <-waitCh:
			fmt.Fprintf(os.Stderr, "wait: %v\n", e)
		default:
			fmt.Fprintf(os.Stderr, "fast exit: %v\n", err)
		}
	}

	code, err := ExitCode(cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Fprintf(os.Stderr, "exit: %d\n", code)

	os.Exit(code)
}
