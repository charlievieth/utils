package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"time"
)

func main() {
	c := exec.Command("pipe_recv")
	c.Stderr = new(bytes.Buffer)
	c.Stdout = new(bytes.Buffer)
	// stdout, err := c.StdoutPipe()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	stdin, err := c.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
	// buf := new(bytes.Buffer)
	tick := time.NewTicker(time.Millisecond * 250)
	for _ = range tick.C {
		_, err := stdin.Write([]byte("hello, ok!"))
		if err != nil {
			fmt.Printf("Write Error (%s): %#v\n", err, err)
			break
		}
		fmt.Println("Ok")
		// buf.Reset()
		// if _, err := buf.ReadFrom(stdout); err != nil {
		// 	fmt.Printf("Read Error: %#v\n", err)
		// 	break
		// }
	}
}
