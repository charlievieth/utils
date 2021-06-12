package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	Path       []string
	SortedPath []string
)

func EnvContains(s string) bool {
	if SortedPath == nil {
		panic("not initialized")
	}
	i := sort.SearchStrings(SortedPath, s)
	return i < len(SortedPath) && SortedPath[i] == s
}

func main() {
	root := &cobra.Command{
		Use: "spathutil",

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			s := os.Getenv("PATH")
			if s == "" {
				return errors.New("PATH is empty")
			}
			Path = strings.Split(s, string(os.PathListSeparator))
			SortedPath = append([]string(nil), Path...)
			sort.Strings(SortedPath)
			return nil
		},
	}

	contains := &cobra.Command{
		Use:  "contains",
		Args: cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			if ok, _ := cmd.Flags().GetBool("quiet"); ok {
				cmd.SetErr(io.Discard)
				cmd.SetOut(io.Discard)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				os.Exit(1)
			}
			all := true
			for _, s := range args {
				ok := EnvContains(s)
				fmt.Fprintf(cmd.OutOrStdout(), "%t\t%s\n", ok, s)
				all = !all || ok
			}
			if !all {
				os.Exit(1)
			}
		},
	}
	contains.Flags().BoolP("quiet", "q", false, "Quiet; don't write anything to stdout.")

	dump := &cobra.Command{
		Use: "dump",
		Run: func(cmd *cobra.Command, args []string) {
			for _, s := range Path {
				fmt.Println(s)
			}
		},
	}

	cmds := []*cobra.Command{
		contains,
		dump,
	}
	root.AddCommand(cmds...)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err)
		os.Exit(1)
	}
}
