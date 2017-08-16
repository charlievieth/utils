package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

var (
	ParentPID  int
	ParentArgs string
	MasterPID  int
)

// Create pid files under a dir like: com.apple.launchd.jyqkehTTrS

func init() {
	flag.IntVar(&ParentPID, "p", -1, "Parent pid")
	flag.StringVar(&ParentArgs, "a", "", "Parent arguments")
	flag.IntVar(&MasterPID, "m", -1, "Master pid")
}

type WaitResponse struct {
	State *os.ProcessState
	Err   error
}

func Wait(pid int, ch chan WaitResponse) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	go func() {
		s, err := p.Wait()
		ch <- WaitResponse{State: s, Err: err}
	}()
	return nil
}

func init() {
	ch := make(chan os.Signal, 100)
	signal.Notify(ch)
	go func() {
		for {
			<-ch
		}
	}()
}

func main() {
	flag.Parse()

	parent := make(chan WaitResponse, 1)
	if err := Wait(ParentPID, parent); err != nil {
		log.Fatalf("parent: %s", err)
	}

	master := make(chan WaitResponse, 1)
	if err := Wait(MasterPID, master); err != nil {
		log.Fatalf("master: %s", err)
	}

	select {
	case <-parent:
	case <-master:
	}
}
