package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

var completionCmd = cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(yourprogram completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ yourprogram completion bash > /etc/bash_completion.d/yourprogram
  # macOS:
  $ yourprogram completion bash > /usr/local/etc/bash_completion.d/yourprogram

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ yourprogram completion zsh > "${fpath[1]}/_yourprogram"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ yourprogram completion fish | source

  # To load completions for each session, execute once:
  $ yourprogram completion fish > ~/.config/fish/completions/yourprogram.fish

PowerShell:

  PS> yourprogram completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> yourprogram completion powershell > yourprogram.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

var root = cobra.Command{
	Use:   "jq",
	Short: "jq shell completion",
	Args:  cobra.ArbitraryArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		exe, err := exec.LookPath("jq")
		if err != nil {
			return err
		}
		// exec JQ in case something needs it for completion
		return syscall.Exec(exe, os.Args[1:], syscall.Environ())
	},
}

func main() {
	flags := root.Flags()

	flags.BoolP("ascii-output", "a", false, "force jq to produce pure ASCII output")
	flags.BoolP("color-output", "C", false, "colorize JSON")
	flags.BoolP("compact-output", "c", false, "compact instead of pretty-printed output")
	flags.BoolP("exit-status", "e", false, "set the exit status code based on the output")
	flags.BoolP("help", "h", false, "print help and exit")
	flags.BoolP("join-output", "j", false, "like -r but jq won't print a newline after each output.")
	flags.BoolP("monochrome-output", "M", false, "monochrome (don't colorize JSON)")
	flags.BoolP("null-input", "n", false, "use `null` as the single input value")
	flags.BoolP("raw-input", "R", false, "read raw strings, not JSON texts")
	flags.BoolP("raw-output", "r", false, "output raw strings, not JSON texts")
	flags.BoolP("slurp", "s", false, "read the entire input stream into a large array and run the filter just once.")
	flags.BoolP("sort-keys", "S", false, "sort keys of objects on output")
	flags.BoolP("version", "V", false, "print version and exit")
	flags.StringP("from-file", "f", "", "read filter from the file rather than from a command line")

	flags.Bool("arg", false, "set variable $a to value <v>")
	flags.Bool("argjson", false, "set variable $a to JSON value <v>")
	flags.Bool("args", false, "remaining arguments are string arguments, not files")
	flags.Bool("debug-dump-disasm", false, "")
	flags.Bool("debug-trace", false, "")
	flags.Bool("debug-trace=all", false, "")
	flags.Bool("jsonargs", false, "remaining arguments are JSON arguments, not files")
	flags.Bool("seq", false, "use the application/json-seq MIME type scheme for separating JSON texts in jq's input and output")
	flags.Bool("stream", false, "parse  the input in streaming fashion")
	flags.Bool("stream-errors", false, "")
	flags.Bool("tab", false, "use tabs for indentation")
	flags.Bool("unbuffered", false, "flush the output after each json object is printed")
	flags.Int("indent", 2, "")
	flags.String("argfile", "", "do not use. use --slurpfile instead")
	flags.String("L", "", "")
	flags.String("rawfile", "", "set variable $a to a string consisting of the contents of <f>")
	flags.String("run-tests", "", "")
	flags.String("slurpfile", "", "slurpfile a f  set variable $a to an array of JSON texts read from <f>")

	root.AddCommand(&completionCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
