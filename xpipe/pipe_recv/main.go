package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Line struct {
	b   []byte
	err error
}

func main() {
	r := bufio.NewReader(os.Stdin)
	line := make(chan Line, 1)
	go func() {
		for {
			b, err := r.ReadBytes('\n')
			line <- Line{b, err}
		}
	}()
	ch := make(chan bool)
	go func() {
		for {
			select {
			case <-ch:
				close(ch)
				return
			case ll := <-line:
				if ll.err != nil {
					fmt.Fprintf(os.Stderr, "Read Error: %+v", ll.err)
					break
				}
				m := map[string]int{
					"Size": len(ll.b),
				}
				if err := json.NewEncoder(os.Stdout).Encode(m); err != nil {
					fmt.Fprintf(os.Stderr, "JSON Error: %+v", err)
					break
				}
			}
		}
	}()
	<-time.Tick(time.Second * 1)
	ch <- true
	<-ch
	fmt.Println("Done")
}
