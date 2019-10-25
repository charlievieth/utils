// subl-completion is a package for generating auto complete
// commands for sublime text.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func SublCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "subl",
		Aliases:    []string{"subl"},
		SuggestFor: []string{"subl"},
	}
	flags := cmd.PersistentFlags()
	flags.SortFlags = false

	flags.String("project", "", "Load the given project")
	cmd.MarkPersistentFlagFilename("project", "sublime-project")

	flags.String("command", "", "Run the given command")

	flags.BoolP("new-window", "n", false, "Open a new window")
	flags.BoolP("add", "a", false, "Add folders to the current window")
	flags.BoolP("wait", "w", false, "Wait for the files to be closed before returning")
	flags.BoolP("background", "b", false, "Don't activate the application")
	flags.BoolP("stay", "s", false, "Keep the application activated after closing the file")
	flags.BoolP("help", "h", false, "Show help (this message) and exit")
	flags.BoolP("version", "v", false, "Show version and exit")

	return cmd
}

func main() {
	err := SublCommand().GenBashCompletion(os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
