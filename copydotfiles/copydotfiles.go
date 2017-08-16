package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func HomeDirectory() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func main() {
	homeDir, err := HomeDirectory()
	if err != nil {
		Fatal(err)
	}
	fmt.Printf("backup directory [%s]:", filepath.Join(homeDir, ".dotfiles.backup"))
	var backupDir string

	if _, err := fmt.Scanln(&backupDir); err != nil {
		Fatal(err)
	}

	fmt.Println(backupDir)
	// fmt.Sscanf("backup: ", format, ...)
}

func Fatal(err interface{}) {
	var s string
	if _, file, line, ok := runtime.Caller(1); ok {
		s = fmt.Sprintf("%s:%d", filepath.Dir(file), line)
	}
	if err != nil {
		switch err.(type) {
		case error, string:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %s\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			}
		default:
			if s != "" {
				fmt.Fprintf(os.Stderr, "Error (%s): %#v\n", s, err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %#v\n", err)
			}
		}
		os.Exit(1)
	}
}
