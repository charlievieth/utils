package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

// [2016-11-01 19:07:12-0400]
func Format(p []byte) []byte {
	const layout = "2006-01-02 15:04:05-0700"
	b := make([]byte, 0, len(layout)+len("[] \n")+len(p))
	b = append(b, '[')
	b = time.Now().AppendFormat(b, layout)
	b = append(b, ']', ' ')
	b = append(b, p...)
	return b
}

func main() {
	var b []byte
	var err error
	in := bufio.NewReader(os.Stdin)
	for err == nil {
		b, err = in.ReadBytes('\n')
		if err != nil {
			break
		}
		if _, werr := os.Stdout.Write(Format(b)); werr != nil {
			fmt.Fprintf(os.Stderr, "write: %s\n", werr)
			break
		}
	}
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "read: %s\n", err)
	}
}
