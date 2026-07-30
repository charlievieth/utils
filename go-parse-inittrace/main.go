package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type Trace struct {
	Module     string
	WallMS     float64
	DurationMS float64
	Bytes      int
	Allocs     int
}

func ParseTrace(line string) (Trace, error) {
	var t Trace
	// init github.com/containerd/containerd/v2/version @0.78 ms, 0.008 ms clock, 0 bytes, 0 allocs
	_, err := fmt.Sscanf(line, "init %s @%f ms, %f ms clock, %d bytes, %d allocs",
		&t.Module, &t.WallMS, &t.DurationMS, &t.Bytes, &t.Allocs)
	return t, err
}

// This simple program parses the output of: GODEBUG=inittrace=1
func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	var traces []Trace
	r := bufio.NewReader(os.Stdin)
	for {
		s, err := r.ReadString('\n')
		if strings.HasPrefix(s, "init ") {
			t, terr := ParseTrace(s)
			if terr != nil {
				log.Println("parse:", err)
			} else {
				traces = append(traces, t)
			}
		}
		if err != nil {
			log.Println("read:", err) // TODO: remove
			break
		}
	}

	PrintJSON(traces)
}

func PrintJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Fatal(err)
	}
}
