package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

func ReadStdin() ([]string, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		return nil, err
	}
	var a []string
	for _, b := range bytes.Fields(buf.Bytes()) {
		a = append(a, string(b))
	}
	return a, nil
}

func realMain() error {
	fields := os.Args[1:]
	if len(os.Args) == 1 {
		var err error
		fields, err = ReadStdin()
		if err != nil {
			return fmt.Errorf("reading stdin: %s", err)
		}
	}
	errCount := 0
	for _, s := range fields {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: parsing (%s): %s\n", s, err)
			errCount++
			continue
		}
		secs := math.Ceil(f)
		t := time.Unix(int64(secs), int64((f-secs)*1e9))
		fmt.Printf("%s: %s\n", s, t.Format(time.RFC3339))
	}
	if errCount > 0 {
		return fmt.Errorf("encountered %d errors", errCount)
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
