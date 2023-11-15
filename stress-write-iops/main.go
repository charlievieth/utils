package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func init() {
	// TODO: consider STDERR so that it doesn't mess up our output
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Lshortfile)
}

type State struct {
	// TODO: this can take a Context
	dirname  string
	next     *atomic.Int64
	complete *atomic.Int64
	errors   *atomic.Int64
	writeCh  chan struct{}
	// cond     *sync.Cond
}

type Deleter struct {
	// state    *State
	ctx      context.Context
	dirname  string
	complete *atomic.Int64 // # completed writes
	last     *atomic.Int64 // last deleted write file
	// writeCh  chan struct{}
}

func (d *Deleter) Run(wg *sync.WaitGroup, writeCh <-chan struct{}) {
	tick := time.NewTicker(time.Millisecond * 10)
	defer tick.Stop()
	done := d.ctx.Done()
	d.last.Store(1) // 1 based
	for {
		select {
		case <-writeCh:
			n := d.complete.Load()
			i := d.last.Load()
			if n-i < 50 {
				break
			}
			for ; i <= n; i++ {
				err := os.Remove(d.dirname + "/tmp." + strconv.FormatInt(i, 10))
				if err != nil {
					log.Println("delete:", err)
				}
			}
			d.last.Store(n + 1)
		case <-done:
			return
		}
	}
}

// func (d *Deleter) Run(wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	tick := time.NewTicker(time.Millisecond)
// 	defer tick.Stop()
// 	done := d.ctx.Done()
// 	d.last.Store(1) // 1 based
// 	for range tick.C {
// 		select {
// 		case <-tick.C:
// 			n := d.complete.Load()
// 			for i := d.last.Load(); i <= n; i++ {
// 				if err := os.Remove(d.dirname + "/tmp." + strconv.FormatInt(i, 10)); err != nil {
// 					log.Println("delete:", err)
// 				}
// 			}
// 			d.last.Store(n + 1)
// 		case <-done:
// 			return
// 		}
// 	}
// }

type Writer struct {
	ctx       context.Context
	dirname   string
	data      []byte
	next      *atomic.Int64
	complete  *atomic.Int64
	writeTime *atomic.Int64
	writeCh   chan<- struct{}
}

func (w *Writer) WriteFile() error {
	name := w.dirname + "/tmp." + strconv.FormatInt(w.next.Add(1), 10)
	t := time.Now()
	err := os.WriteFile(name, w.data, 0644)
	w.writeTime.Add(int64(time.Since(t)))
	w.complete.Add(1)
	select {
	case w.writeCh <- struct{}{}:
	default:
	}
	return err
}

func (w *Writer) Run(wg *sync.WaitGroup, interval time.Duration) {
	defer wg.Done()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	done := w.ctx.Done()
	for {
		select {
		case <-tick.C:
			if err := w.WriteFile(); err != nil {
				log.Println("write:", err)
			}
		case <-done:
			return
		}
	}
}

type Delta struct {
	prev, curr int64
}

func (d *Delta) Delta(new int64) int64 {
	return new - d.prev
}

func (d *Delta) Rate(new int64, dur time.Duration) float64 {
	return float64(d.Delta(new)) / dur.Seconds()
}

func realMain() error {
	numWriters := flag.Int("n", runtime.NumCPU(), "number of concurrent writers")
	writeInterval := flag.Duration("d", time.Millisecond/4, "writer interval")
	flag.Parse()

	if *numWriters <= 0 || *writeInterval <= 0 {
		flag.Usage()
		log.Fatalf("invalid argument: numWriters: %d writeInterval: %d",
			*numWriters, *writeInterval)
	}

	tmpdir, err := os.MkdirTemp("", "stress-write-iops-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	log.Println("tempdir:", tmpdir)

	// Create a rescue file that we can delete if we run out of disk space.
	if err := Fallocate("rescue_file.dat", 64*1024*1024); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCtx, sigStop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-ctx.Done():
		case <-sigCtx.Done():
			select {
			case <-ctx.Done():
			default:
			}
			log.Println("signaled: stopping now...")
			cancel()  // parent context
			sigStop() // stop signal handling
		}
	}()

	var (
		wg       sync.WaitGroup
		next     atomic.Int64
		complete atomic.Int64
	)

	writeCh := make(chan struct{}, 1)
	d := &Deleter{
		ctx:      ctx,
		dirname:  tmpdir,
		complete: &complete,
		last:     new(atomic.Int64),
	}
	wg.Add(1)
	go d.Run(&wg, writeCh)

	for i := 0; i < *numWriters; i++ {
		w := &Writer{
			ctx:       ctx,
			dirname:   tmpdir,
			data:      make([]byte, 4096), // TODO: make this harder
			next:      &next,
			complete:  &complete,
			writeCh:   writeCh,
			writeTime: new(atomic.Int64),
		}
		wg.Add(1)
		go w.Run(&wg, *writeInterval)
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		tick := time.NewTicker(time.Second)
		last := time.Now()
		var prevNext int64
		var prevComplete int64
		_ = prevNext
		// fmt.Println("complete:\t next: ")
		for {
			select {
			case now := <-tick.C:
				c := complete.Load()
				n := next.Load()
				queue := complete.Load() - d.last.Load()
				rateC := float64(c-prevComplete) / now.Sub(last).Seconds()
				fmt.Printf("complete: %.2f/s\tqueue: %d\t\n", rateC, queue)
				prevComplete = c
				prevNext = n
				last = now
			case <-ctx.Done():
				return
			}
		}
	}(&wg)

	wg.Wait()
	return nil
}

func main() {
	if err := realMain(); err != nil {
		log.Println("failed:", err)
		os.Exit(1)
	}
}
