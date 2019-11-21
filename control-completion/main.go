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

func AddAncestorsCmd() *cobra.Command {
	// positional arguments:
	//   image       the image name
	//   version     the image version
	cmd := &cobra.Command{
		Use: "ancestors",
	}
	return cmd
}

func AddBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "build",
		Args: ImageArg,
	}
	flags := cmd.LocalFlags()
	flags.Bool("download-tar", false, "")
	flags.String("sub-image", "", "")
	return cmd
}

func AddCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "clean",
	}
	flags := cmd.LocalFlags()
	flags.StringP("preserve-overrides", "p", "", "")
	return cmd
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

func AddContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "context",
	}
	cmd.AddCommand(&cobra.Command{
		Use:  "list",
		Args: cobra.NoArgs,
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "use",
		Args: ContextArg(),
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "create",
		Args: ContextArg(),
	})
	cmd.AddCommand(&cobra.Command{
		Use:  "delete",
		Args: ContextArg(),
	})
	// Add completion aliases
	for _, c := range cmd.Commands() {
		c.Aliases = append(c.Aliases, c.Use)
		c.SuggestFor = append(c.SuggestFor, c.Use)
		cmd.ValidArgs = append(cmd.ValidArgs, c.Use)
	}
	return cmd
}

func AddDeployCmd() *cobra.Command {
	// WARN: ignoring this
	cmd := &cobra.Command{
		Use: "deploy",
	}
	return cmd
}

func AddDiscoverCmd() *cobra.Command {
	// WARN: ignoring this
	cmd := &cobra.Command{
		Use: "discover",
	}
	return cmd
}

func AddEnsureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ensure",
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "") // TODO: Add completion for this
	flags.BoolP("all", "a", false, "")
	flags.BoolP("only", "o", false, "")
	flags.Bool("json", false, "")
	return cmd
}

func AddEnterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "enter",
		Args: ContainerArg,
	}
	return cmd
}

func AddExtractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "extract",
		Args: func(cmd *cobra.Command, args []string) error {
			// positional arguments:
			//   container  the container to extract from
			//   path       the source path
			//   dest       the destination path
			return cobra.ExactArgs(3)(cmd, args)
		},
	}
	flags := cmd.LocalFlags()
	flags.StringP("image", "i", "", "") // TODO: Add completion for this
	flags.StringP("group", "g", "", "") // TODO: Add completion for this
	flags.StringP("test", "t", "", "")
	flags.StringP("version", "v", "", "")
	flags.StringP("sub-image", "s", "", "")
	flags.BoolP("container-only", "c", false, "")
	return cmd
}

func AddKillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "kill",
	}
	flags := cmd.LocalFlags()
	flags.StringP("id", "i", "", "")
	flags.StringP("signal", "s", "", "")
	return cmd
}

func AddListCmd() *cobra.Command {
	// Examples:
	//     control list
	//     control list images
	//     control list groups
	//     control list simulatedrides.qa
	cmd := &cobra.Command{
		Use:  "list",
		Args: cobra.MaximumNArgs(1),
	}
	return cmd
}

func AddLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "logs",
		Args: ContainerArg,
	}
	flags := cmd.LocalFlags()
	flags.String("filepath", "", "")
	flags.IntP("lines", "n", 0, "")
	flags.StringP("pattern", "p", "", "")
	flags.BoolP("follow", "f", false, "")
	flags.BoolP("insensitive-case", "i", false, "")
	return cmd
}

func AddManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "manifest",
		Args: ImageArg,
	}
	return cmd
}

func AddPublishManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "publish-manifest",
	}
	return cmd
}

func AddOverrideCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "override",
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
	return cmd
}

func AddProvisionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "provision",
		Args: ContainerArg,
	}
	return cmd
}

func AddPullCmd() *cobra.Command {
	// positional arguments:
	//   image    the image name
	//   version  the image version
	cmd := &cobra.Command{
		Use:  "pull",
		Args: cobra.MaximumNArgs(2),
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.StringP("registry-url", "r", "", "")
	return cmd
}

func AddPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "push",
	}
	flags := cmd.LocalFlags()
	flags.Bool("nomanifest", false, "")
	flags.String("regions", "", "")
	flags.StringP("group", "g", "", "")
	flags.StringP("cluster-name", "c", "", "")
	return cmd
}

func AddPushManifestCmd() *cobra.Command {
	// positional arguments:
	//   name     the image name
	//   version  the image version
	cmd := &cobra.Command{
		Use:  "push-manifest",
		Args: cobra.ExactArgs(2),
	}
	return cmd
}

func AddRdependsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "rdepends",
		Args: ImageArg,
	}
	return cmd
}

// skipping
func AddRegisterHostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "register-host",
	}
	return cmd
}

// skipping
func AddReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "release",
	}
	return cmd
}

func AddResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "reset",
	}
	flags := cmd.LocalFlags()
	flags.Bool("no-clean", false, "")
	return cmd
}

func AddRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "restart",
		Args: ContainerArg,
	}
	return cmd
}

func AddRunCmd() *cobra.Command {
	// positional arguments:
	//   container  the container to run
	//   subargs    arguments passed to command
	cmd := &cobra.Command{
		Use:  "run",
		Args: cobra.MinimumNArgs(1),
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
	return cmd
}

// skipping
func AddSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "snapshot",
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	return cmd
}

func AddStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "start",
		Args: ContainerArg,
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.Bool("json", false, "")
	return cmd
}

func AddStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "stats",
	}
	flags := cmd.LocalFlags()
	flags.BoolP("post", "p", false, "")
	flags.StringP("failure", "f", "", "")
	return cmd
}

func AddStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"ps"},
		Args:    ContainerOrGroupArg, // optional
	}
	flags := cmd.LocalFlags()
	flags.BoolP("all", "a", false, "")
	flags.BoolP("time", "t", false, "")
	flags.BoolP("verbose", "v", false, "")
	return cmd
}

func AddStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "stop",
		Args: ContainerArg, // optional
	}
	flags := cmd.LocalFlags()
	flags.StringP("group", "g", "", "")
	flags.BoolP("all", "a", false, "")
	return cmd
}

func AddTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
	}
	flags := cmd.LocalFlags()
	flags.BoolP("debug", "d", false, "")
	flags.Bool("extract-logs", false, "") // removed "el"
	flags.StringP("bracket-id", "b", "", "")
	flags.StringP("environment", "e", "", "")
	flags.String("archive-url", "", "")
	return cmd
}

func AddTreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "tree",
		Args: ContainerOrGroupArg,
	}
	return cmd
}

func AddUpdateCmd() *cobra.Command {
	// positional arguments:
	//   image  parent image
	//   child  child images to update
	cmd := &cobra.Command{
		Use:  "update",
		Args: cobra.MaximumNArgs(2),
	}
	flags := cmd.LocalFlags()
	flags.StringP("author", "a", "", "")
	flags.StringP("reason", "r", "", "")
	flags.StringP("sleep", "s", "", "")
	flags.BoolP("merge", "m", false, "")
	flags.StringP("revision", "R", "", "")
	flags.StringP("env", "e", "", "")
	flags.StringP("branch-name", "b", "", "")
	return cmd
}

func AddUrlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "url",
		Args: ContainerArg,
	}
	flags := cmd.LocalFlags()
	flags.BoolP("public", "p", false, "")
	flags.BoolP("local", "l", false, "")
	return cmd
}

func AddVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "version",
		Args: ImageArg,
	}
	flags := cmd.LocalFlags()
	flags.IntP("limit", "l", 0, "")
	return cmd
}

func AddSubCmds(root *cobra.Command) *cobra.Command {
	cmds := []func() *cobra.Command{
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
	for _, fn := range cmds {
		cmd := fn()
		cmd.Run = func(*cobra.Command, []string) {}
		cmd.LocalFlags().SortFlags = false
		root.AddCommand(cmd)
		root.ValidArgs = append(root.ValidArgs, cmd.Use)
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
