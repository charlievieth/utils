// subl-completion is a package for generating auto complete
// commands for sublime text.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func ImageArg(cmd *cobra.Command, args []string) error {
	return cobra.ExactArgs(1)(cmd, args)
}

func ContainerArg(cmd *cobra.Command, args []string) error {
	return cobra.ExactArgs(1)(cmd, args)
}

func ContainerOrGroupArg(cmd *cobra.Command, args []string) error {
	return cobra.ExactArgs(1)(cmd, args)
}

func AddAncestorsCmd(root *cobra.Command) {
	// positional arguments:
	//   image       the image name
	//   version     the image version
	cmd := &cobra.Command{
		Use: "ancestors",
		Run: func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddBuildCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "build",
		Args: ImageArg,
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.Bool("download-tar", false, "")
	flags.String("sub-image", "", "")
	root.AddCommand(cmd)
}

func AddCleanCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "clean",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("preserve-overrides", "p", "", "")
	root.AddCommand(cmd)
}

/*
   parser_use = subparsers.add_parser('use')
   parser_use.add_argument('context', help='context to switch to')
   parser_use.set_defaults(func=use_context)

   parser_use = subparsers.add_parser('create')
   parser_use.add_argument('context', help='context to create')
   parser_use.set_defaults(func=create_context)

   parser_use = subparsers.add_parser('delete')
   parser_use.add_argument('context', help='context to delete')
   parser_use.set_defaults(func=delete_environment)

*/

func ContextArg() cobra.PositionalArgs {
	return cobra.ExactArgs(1)
}

func AddContextCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "context",
		Run: func(*cobra.Command, []string) {},
	}
	cmd.AddCommand(&cobra.Command{
		Use:  "list",
		Args: cobra.NoArgs,
		Run:  func(*cobra.Command, []string) {},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "use",
		Args: ContextArg(),
		Run:  func(*cobra.Command, []string) {},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "create",
		Args: ContextArg(),
		Run:  func(*cobra.Command, []string) {},
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete",
		Args: ContextArg(),
		Run:  func(*cobra.Command, []string) {},
	})
	// Add completion aliases
	for _, c := range cmd.Commands() {
		c.Aliases = append(c.Aliases, c.Use)
		c.SuggestFor = append(c.SuggestFor, c.Use)
	}
	root.AddCommand(cmd)
}

func AddDeployCmd(root *cobra.Command) {
	// WARN: ignoring this
	cmd := &cobra.Command{
		Use: "deploy",
		Run: func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddDiscoverCmd(root *cobra.Command) {
	// WARN: ignoring this
	cmd := &cobra.Command{
		Use: "discover",
		Run: func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddEnsureCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "ensure",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "") // TODO: Add completion for this
	flags.BoolP("all", "a", false, "")
	flags.BoolP("only", "o", false, "")
	flags.Bool("json", false, "")
	root.AddCommand(cmd)
}

func AddEnterCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "enter",
		Args: ContainerArg,
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddExtractCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "extract",
		Args: func(cmd *cobra.Command, args []string) error {
			// positional arguments:
			//   container  the container to extract from
			//   path       the source path
			//   dest       the destination path
			return cobra.ExactArgs(3)(cmd, args)
		},
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("image", "i", "", "") // TODO: Add completion for this
	flags.StringP("group", "g", "", "") // TODO: Add completion for this
	flags.StringP("test", "t", "", "")
	flags.StringP("version", "v", "", "")
	flags.StringP("sub-image", "s", "", "")
	flags.BoolP("container-only", "c", false, "")
	root.AddCommand(cmd)
}

func AddKillCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "kill",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("id", "i", "", "")
	flags.StringP("signal", "s", "", "")
	root.AddCommand(cmd)
}

func AddListCmd(root *cobra.Command) {
	// Examples:
	//     control list
	//     control list images
	//     control list groups
	//     control list simulatedrides.qa
	cmd := &cobra.Command{
		Use:  "list",
		Args: cobra.MaximumNArgs(1),
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddLogsCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "logs",
		Args: ContainerArg,
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.String("filepath", "", "")
	flags.IntP("lines", "n", 0, "")
	flags.StringP("pattern", "p", "", "")
	flags.BoolP("follow", "f", false, "")
	flags.BoolP("insensitive-case", "i", false, "")
	root.AddCommand(cmd)
}

func AddManifestCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "manifest",
		Args: ImageArg,
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddPublishManifestCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "publish-manifest",
		Run: func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddOverrideCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "override",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.BoolP("list", "l", false, "")
	flags.StringP("image", "i", "", "") // TODO: required for some commands
	flags.StringP("group", "g", "", "") // TODO: required for some commands
	flags.BoolP("read-env", "e", false, "")
	flags.StringP("ref", "r", "", "")
	flags.String("service-release", "", "")
	flags.BoolP("clear", "c", false, "")
	flags.BoolP("unset", "u", false, "") // TODO: Complete image
	root.AddCommand(cmd)
}

func AddProvisionCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "provision",
		Args: ContainerArg,
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddPullCmd(root *cobra.Command) {
	// positional arguments:
	//   image    the image name
	//   version  the image version
	cmd := &cobra.Command{
		Use:  "pull",
		Args: cobra.MaximumNArgs(2),
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.StringP("registry-url", "r", "", "")
	root.AddCommand(cmd)
}

func AddPushCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "push",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.Bool("nomanifest", false, "")
	flags.String("regions", "", "")
	flags.StringP("group", "g", "", "")
	flags.StringP("cluster-name", "c", "", "")
	root.AddCommand(cmd)
}

func AddPushManifestCmd(root *cobra.Command) {
	// positional arguments:
	//   name     the image name
	//   version  the image version
	cmd := &cobra.Command{
		Use:  "push-manifest",
		Args: cobra.ExactArgs(2),
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddRdependsCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "rdepends",
		Args: ImageArg,
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

// skipping
func AddRegisterHostCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "register-host",
		Run: func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

// skipping
func AddReleaseCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "release",
		Run: func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddResetCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "reset",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.Bool("no-clean", false, "")
	root.AddCommand(cmd)
}

func AddRestartCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "restart",
		Args: ContainerArg,
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddRunCmd(root *cobra.Command) {
	// positional arguments:
	//   container  the container to run
	//   subargs    arguments passed to command
	cmd := &cobra.Command{
		Use:  "run",
		Args: cobra.MinimumNArgs(1),
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.StringP("iam-role", "r", "", "")
	flags.StringP("credentials", "t", "", "")
	flags.StringP("environment", "e", "", "")
	flags.StringP("bracket-id", "b", "", "")
	flags.Bool("background", false, "")
	flags.StringP("debug", "d", "", "")
	flags.StringP("command", "c", "", "")
	root.AddCommand(cmd)
}

// skipping
func AddSnapshotCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "snapshot",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	root.AddCommand(cmd)
}

func AddStartCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "start",
		Args: ContainerArg,
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.Bool("json", false, "")
	root.AddCommand(cmd)
}

func AddStatsCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "stats",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.BoolP("post", "p", false, "")
	flags.StringP("failure", "f", "", "")
	root.AddCommand(cmd)
}

func AddStatusCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"ps"},
		Args:    ContainerOrGroupArg, // optional
		Run:     func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.BoolP("all", "a", false, "")
	flags.BoolP("time", "t", false, "")
	flags.BoolP("verbose", "v", false, "")
	root.AddCommand(cmd)
}

func AddStopCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "stop",
		Args: ContainerArg, // optional
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.BoolP("all", "a", false, "")
	root.AddCommand(cmd)
}

func AddTestCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use: "test",
		Run: func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.BoolP("debug", "d", false, "")
	flags.Bool("extract-logs", false, "") // removed "el"
	flags.StringP("bracket-id", "b", "", "")
	flags.StringP("environment", "e", "", "")
	flags.String("archive-url", "", "")
	root.AddCommand(cmd)
}

func AddTreeCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "tree",
		Args: ContainerOrGroupArg,
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(cmd)
}

func AddUpdateCmd(root *cobra.Command) {
	// positional arguments:
	//   image  parent image
	//   child  child images to update
	cmd := &cobra.Command{
		Use:  "update",
		Args: cobra.MaximumNArgs(2),
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.StringP("author", "a", "", "")
	flags.StringP("reason", "r", "", "")
	flags.StringP("sleep", "s", "", "")
	flags.BoolP("merge", "m", false, "")
	flags.StringP("revision", "R", "", "")
	flags.StringP("env", "e", "", "")
	flags.StringP("branch-name", "b", "", "")
	root.AddCommand(cmd)
}

func AddUrlCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "url",
		Args: ContainerArg,
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.BoolP("public", "p", false, "")
	flags.BoolP("local", "l", false, "")
	root.AddCommand(cmd)
}

func AddVersionCmd(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:  "version",
		Args: ImageArg,
		Run:  func(*cobra.Command, []string) {},
	}
	flags := cmd.LocalFlags()
	flags.IntP("limit", "l", 0, "")
	root.AddCommand(cmd)
}

func AddSubCmds(root *cobra.Command) *cobra.Command {
	cmds := []func(*cobra.Command){
		AddAncestorsCmd,
		AddBuildCmd,
		AddCleanCmd,
		AddContextCmd,
		AddDeployCmd,
		AddDiscoverCmd,
		AddEnsureCmd,
		AddEnterCmd,
		AddExtractCmd,
		AddKillCmd,
		AddListCmd,
		AddLogsCmd,
		AddManifestCmd,
		AddPublishManifestCmd,
		AddOverrideCmd,
		AddProvisionCmd,
		AddPullCmd,
		AddPushCmd,
		AddPushManifestCmd,
		AddRdependsCmd,
		AddRegisterHostCmd,
		AddReleaseCmd,
		AddResetCmd,
		AddRestartCmd,
		AddRunCmd,
		AddSnapshotCmd,
		AddStartCmd,
		AddStatsCmd,
		AddStatusCmd,
		AddStopCmd,
		AddTestCmd,
		AddTreeCmd,
		AddUpdateCmd,
		AddUrlCmd,
		AddVersionCmd,
	}
	for _, cmd := range cmds {
		cmd(root)
	}
	return root
}

func SublCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "control",
		Aliases:    []string{"control", "local_control", "lc"},
		SuggestFor: []string{"control", "local_control", "lc"},
	}
	cmd = AddSubCmds(cmd)

	flags := cmd.PersistentFlags()
	flags.SortFlags = false

	flags.BoolP("help", "h", false, "show this help message and exit")
	flags.BoolP("verbose", "v", false, "verbose output")
	flags.BoolP("quiet", "q", false, "quieter output")

	flags.Bool("kubernetes", false, "enable experimental direct Kubernetes support")
	flags.Bool("devboxless", false, "enable experimental devbox-less operation")

	flags.StringP("namespace", "n", "", "Kubernetes namespace for resources")                     // bind
	flags.String("context", "", "Devkube working context. Overrides all other context settings.") // bind

	flags.Bool("enabledevdns", false, "enable the development dns server to be used on Kubernetes")
	flags.Bool("enablek8sconfidant", false, "enable k8sconfidant secret injector for Kubernetes")
	flags.Bool("enableloggingngsidecar", false, "Logging-ng for Devkube. Will inject fluentbit sidecar for all service containers")

	return cmd
}

func main() {
	// cmd := SublCommand()
	// fmt.Println(cmd.Execute())
	// return
	err := SublCommand().GenBashCompletionFile("out.sh")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
