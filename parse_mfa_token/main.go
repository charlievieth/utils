package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type MFAToken struct {
	RoleArn    string
	Expiration string
}

func printPS1(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	var tok MFAToken
	if err := json.NewDecoder(f).Decode(&tok); err != nil {
		return err
	}
	const layout = "20060102T150405Z07"
	t, err := time.Parse(layout, tok.Expiration)
	if err != nil {
		return err
	}
	fmt.Printf("%s (%d min)", tok.RoleArn, int64(time.Until(t).Minutes()))
	return nil
}

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		return // fail silently
	}
	if err := printPS1(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
