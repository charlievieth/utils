package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func main() {
	cmd := cobra.Command{
		Use: "relpath: PATH...",
	}
	resolveLinks := cmd.Flags().BoolP("resolve-links", "r", false,
		"resolve symlinks in the relative path")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		for _, s := range args {
			p, err := filepath.Rel(wd, s)
			if err != nil {
				return err
			}
			if *resolveLinks {
				p, err = filepath.EvalSymlinks(p)
				if err != nil {
					return err
				}
			}
			fmt.Println(p)
		}
		return nil
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
