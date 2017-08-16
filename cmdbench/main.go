package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "%s [utility [argument ...]]", os.Args[0])
	os.Exit(1)
}

type readCounter struct {
	n int64
}

func (r *readCounter) Read(p []byte) (int, error) {
	r.n += int64(len(p))
	return len(p), nil
}

type writeCounter struct {
	n int64
}

func (w *writeCounter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

func format(n int64, d time.Duration) string {
	kb := float64(n) / 1024
	mb := float64(n) / (1024 * 1024)
	kbs := kb / d.Seconds()
	mbs := mb / d.Seconds()
	return fmt.Sprintf("Kb: %.2f\tKb/s: %.2f\tMb: %.2f\tMb/s: %.2f", kb, kbs, mb, mbs)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	name := os.Args[1]
	args := os.Args[2:]

	var stdout writeCounter
	var stderr writeCounter

	cmd := exec.Command(name, args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	t := time.Now()
	if err := cmd.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	d := time.Since(t)

	fmt.Fprint(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "user:\t%s\n", cmd.ProcessState.UserTime())
	fmt.Fprintf(os.Stderr, "sys:\t%s\n", cmd.ProcessState.SystemTime())
	fmt.Fprintf(os.Stderr, "stdout:\t%s\n", format(stdout.n, d))
	fmt.Fprintf(os.Stderr, "stderr:\t%s\n", format(stderr.n, d))
}
