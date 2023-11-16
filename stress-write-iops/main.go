package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

func init() {
	// TODO: consider STDERR so that it doesn't mess up our output
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Lshortfile)
}

type Writer struct {
	ctx        context.Context
	next       *atomic.Int64
	complete   *atomic.Int64
	writeTime  *atomic.Int64
	writeBytes *atomic.Int64
	numWriters int64
	rr         *rand.Rand
	data       []byte
	dirname    string
	syncFiles  bool // optionally flush files to disk (SLOW!!!)
}

func (w *Writer) WriteFile(name string) error {
	if w.rr.Intn(100) <= 10 {
		os.ReadFile(name)
	}
	if w.rr.Intn(100) <= 10 {
		if _, err := os.Lstat(name); err == nil {
			_ = os.Remove(name) // ignore error
		}
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	exit := func(err error) error {
		f.Close()
		os.Remove(name)
		return err
	}

	// Write random sections
	for i := 0; i < 4; i++ {
		sz := int64(len(w.data))
		off := w.rr.Int63n(sz)
		n := w.rr.Int63n(sz)
		if off+n > sz {
			n = sz - off
		}
		t := time.Now()
		if _, err := f.WriteAt(w.data[:n], off); err != nil {
			return exit(err)
		}
		// TODO: sync after each write?
		w.writeTime.Add(int64(time.Since(t)))
		w.writeBytes.Add(n)
		w.complete.Add(1) // WARN: this is now write count
	}

	if w.syncFiles {
		if err := f.Sync(); err != nil {
			return exit(err)
		}
	}
	if err := f.Close(); err != nil {
		return exit(err)
	}
	return nil
}

func (w *Writer) Run(wg *sync.WaitGroup, interval time.Duration) {
	defer wg.Done()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			n := w.next.Add(1) % (w.numWriters * 4)
			name := w.dirname + "/tmp." + strconv.FormatInt(n, 10)
			if err := w.WriteFile(name); err != nil {
				log.Println("write:", err)
			}
		case <-w.ctx.Done():
			return
		}
	}
}

func dirSize(name string) (int64, error) {
	// Since we're constantly thrashing this directory
	// the size is an estimate.
	des, err := os.ReadDir(name)
	if err != nil {
		return 0, err
	}
	var size int64
	for _, d := range des {
		if !d.Type().IsRegular() {
			continue
		}
		// Ignore errors here
		if fi, _ := d.Info(); fi != nil {
			size += fi.Size()
		}
	}
	return size, nil
}

// TODO: rename
func printAgentDiskLatencyStats(ctx context.Context, wg *sync.WaitGroup, tmpdir string, interval time.Duration) {
	defer wg.Done()

	// This is slow so update infrequently
	tmpSize := new(atomic.Int64)
	go func() {
		sz, err := dirSize(tmpdir)
		if err != nil && !os.IsNotExist(err) {
			log.Println("dirSize:", err)
		}
		tmpSize.Store(sz)
		tick := time.NewTicker(10 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				sz, err := dirSize(tmpdir)
				if err != nil && !os.IsNotExist(err) {
					log.Println("dirSize:", err)
					continue
				}
				tmpSize.Store(sz)
			case <-ctx.Done():
				return
			}
		}
	}()

	// TODO: use a struct for deltas to make our lives easier
	var (
		pRTime      uint64
		pRCount     uint64
		pWTime      uint64
		pWCount     uint64
		maxRLatency float64
		maxWLatency float64
	)

	tick := time.NewTicker(interval)
	defer tick.Stop()
	first := true
	for {
		select {
		case <-tick.C:
			counters, err := disk.IOCountersWithContext(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Println("error: IOCountersWithContext:", err)
				}
				return
			}
			var rtime, wtime uint64
			var rcount, wcount uint64
			for _, c := range counters {
				rtime += c.ReadTime
				wtime += c.WriteTime
				rcount += c.ReadCount
				wcount += c.WriteCount
			}
			if first {
				pRTime = rtime
				pWTime = wtime
				pRCount = rcount
				pWCount = wcount
				first = false
				break
			}
			drtime := rtime - pRTime
			dwtime := wtime - pWTime
			drcount := rcount - pRCount
			dwcount := wcount - pWCount
			wLatency := float64(dwtime) / float64(dwcount)
			if wLatency > maxWLatency {
				maxWLatency = wLatency
			}
			rLatency := float64(drtime) / float64(drcount)
			if rLatency > maxRLatency {
				maxRLatency = rLatency
			}
			size := float64(tmpSize.Load()) / (1024 * 1024)
			fmt.Printf("# latency: write: %.2fms read: %.2fms max_write: %.2fms max_read: %.2fms temp_size: %.2fMiB\n",
				wLatency, rLatency, maxWLatency, maxRLatency, size)
		case <-ctx.Done():
			return
		}
	}
}

func realMain() error {
	flag.Usage = func() {
		const msg = "Usage of %s:\n" +
			"Stress disk IO and print latency statistics.\n" +
			"\n" +
			"NOTE: use the ./run.bash script for best results.\n" +
			"\n" +
			"Example:\n" +
			"  $ ./run.bash  -n 256 -d 1ns -sync -disk-stat-int 1s\n" +
			"\n" +
			"Flags:\n"
		fmt.Fprintf(flag.CommandLine.Output(), msg, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	numWriters := flag.Int("n", runtime.NumCPU()*4, "number of concurrent writers")
	writeInterval := flag.Duration("d", time.Nanosecond, "writer interval")
	syncFiles := flag.Bool("sync", false, "force completion of pending disk writes (flush cache)")
	syncInterval := flag.Duration("sync-int", 100*time.Millisecond, "sync interval")
	diskStatInterval := flag.Duration("disk-stat-int", 10*time.Second,
		"collect and print agent style disk latency stats at this interval")
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

	// Print some useful info
	log.Println("# NUM_WRITERS: ", *numWriters)
	log.Println("# TEMPDIR:     ", tmpdir)
	log.Println("# RESCUE_FILE:  /tmp/rescue_file.dat")

	// Create a rescue file that we can delete if we run out of disk space.
	if err := Fallocate("/tmp/rescue_file.dat", 64*1024*1024); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// On CTRL-C cleanup and exit
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
		wg         sync.WaitGroup
		next       atomic.Int64
		complete   atomic.Int64
		writeTime  atomic.Int64
		writeBytes atomic.Int64
	)

	// WARN: careful with size
	writeData := bytes.Repeat([]byte("0123"), (4*1024*1024)/len("0123"))
	numFiles := int64(*numWriters)
	if numFiles < 8 {
		numFiles = 8
	}
	for i := 0; i < *numWriters; i++ {
		w := &Writer{
			ctx:        ctx,
			dirname:    tmpdir,
			data:       writeData, // TODO: create local copy?
			next:       &next,
			complete:   &complete,
			writeTime:  &writeTime,
			writeBytes: &writeBytes,
			numWriters: numFiles,
			rr:         rand.New(rand.NewSource(time.Now().UnixNano())),
			syncFiles:  true, // TOOD: this should be configurable
		}
		wg.Add(1)
		go w.Run(&wg, *writeInterval)
	}

	if *syncFiles {
		go func() {
			// No need for a WaitGroup here
			tick := time.NewTicker(*syncInterval)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					syscall.Sync()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Print agent style disk stats
	wg.Add(1)
	go printAgentDiskLatencyStats(ctx, &wg, tmpdir, *diskStatInterval)

	wg.Wait()
	if err := os.RemoveAll(tmpdir); err != nil && !os.IsNotExist(err) {
		log.Println("cleanup:", err)
		return err
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		log.Println("failed:", err)
		os.Exit(1)
	}
}
