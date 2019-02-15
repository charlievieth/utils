package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/tools/benchmark/parse"
)

func xmain() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "%s: USAGE: TIMESTAMP_NS...\n",
			filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	for _, s := range os.Args[1:] {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error (%s): %s\n", s, err)
			continue
		}
		fmt.Printf("%d: %s\n", n, time.Duration(n))
	}
}

func String(b *parse.Benchmark) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s %d", b.Name, b.N)
	if (b.Measured & parse.NsPerOp) != 0 {
		// fmt.Fprintf(buf, " %.2f ns/op", b.NsPerOp)
		fmt.Fprintf(buf, " %s", time.Duration(b.NsPerOp))
	}
	if (b.Measured & parse.MBPerS) != 0 {
		fmt.Fprintf(buf, " %.2f MB/s", b.MBPerS)
	}
	if (b.Measured & parse.AllocedBytesPerOp) != 0 {
		fmt.Fprintf(buf, " %d B/op", b.AllocedBytesPerOp)
	}
	if (b.Measured & parse.AllocsPerOp) != 0 {
		fmt.Fprintf(buf, " %d allocs/op", b.AllocsPerOp)
	}
	return buf.String()
}

func realMain() error {
	// set, err := parse.ParseSet(os.Stdin)
	// if err != nil {
	// 	return err
	// }
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		text := scan.Text()
		if !strings.HasPrefix(text, "Benchmark") {
			fmt.Println(text)
			continue
		}
		b, err := parse.ParseLine(text)
		if err != nil {
			return err
		}
		fmt.Println(String(b))
	}
	return nil
}

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
