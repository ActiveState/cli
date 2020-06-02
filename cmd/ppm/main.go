package main

// Shim command for the deprecated PPM tool
//
// The tool mimics the top-level sub-command architecture of the deprecated PPM tool:
// http://docs.activestate.com/activeperl/5.24/perl/bin/ppm.html
// If available it prints a suggestion on which state tool command to explore, when the user
// requests a deprecated ppm command.
// Several c

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/spf13/cobra"
)

const docsBaseURI = "https://docs.activestate.com/platform"

const headerMessage = `The Perl package manager is no longer supported.
To manage your Perl packages from the command line, use the state tool instead.
You can find more information on the state tool at
https://www.activestate.com/products/platform/state-tool/
`

func printDefault() {
	fmt.Println(headerMessage)
}

func printVersion() {
	fmt.Println(strings.TrimSpace(fmt.Sprintf(`
%s
PPM shim for ActiveState CLI version %s
Copyright (c) 2020, ActiveState Software
`, headerMessage, constants.Version)))

}

func printSuggestion(ppmIntent, newCommand, docLink string) {
	fmt.Println(strings.TrimSpace(fmt.Sprintf(`
%s

To %s, run:

   %s
   
See "%s/%s" for details.
`, headerMessage, ppmIntent, newCommand, docsBaseURI, docLink)))
}

func addPackagesCommands(cmds []*cobra.Command) []*cobra.Command {
	return append(cmds,
		&cobra.Command{
			Use:   "install",
			Short: "installs new packages",
			Run: func(cmd *cobra.Command, args []string) {
				printSuggestion("add new packages to your project", "state packages add", "state/packages.html")
			},
		},
		&cobra.Command{
			Use:   "upgrade",
			Short: "upgrades installed packages",
			Run: func(cmd *cobra.Command, args []string) {
				printSuggestion("upgrade an existing package", "state packages update", "state/packages.html")
			},
		},
		&cobra.Command{
			Use:   "remove",
			Short: "removes installed packages",
			Run: func(cmd *cobra.Command, args []string) {
				printSuggestion("remove a package from your project", "state packages remove", "state/packages.html")
			},
		},
	)
}

func addVersionCommand(cmds []*cobra.Command) []*cobra.Command {
	return append(cmds,
		&cobra.Command{
			Use:   "version",
			Short: "print version info",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printVersion()
			},
		},
	)
}

func addProjectCommands(cmds []*cobra.Command) []*cobra.Command {
	return append(cmds,
		&cobra.Command{
			Use:   "area",
			Short: "organizes packages in different areas",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(strings.TrimSpace(fmt.Sprintf(`
%s
To manage several projects with varying dependencies, you can organize them as
projects on the platform: %s/projects.html.

To fork an existing project, run:
	state fork
See %s/state/fork.html for details.
	
To create a new project, run:
	state init
See %s/state/init.html for details.
`, headerMessage, docsBaseURI, docsBaseURI, docsBaseURI)))
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "lists installed packages",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printSuggestion("list the packages installed for a project", "state packages", "state/packages.html")
			},
		},
		&cobra.Command{
			Use:   "files",
			Short: "lists the full path name of the files belonging to the given package, one line per file.",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
		&cobra.Command{
			Use:   "verify",
			Short: "checks that the installed files are present and unmodified.",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
	)
}

func addRepositoryCommands(cmds []*cobra.Command) []*cobra.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		&cobra.Command{
			Use:   "repo",
			Short: "manages package repositories",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
		&cobra.Command{
			Use:   "search",
			Short: "searches for packages in all enabled repositories",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printSuggestion("search for a package on the platform", "state packages search", "state/packages.html")
			},
		},
		&cobra.Command{
			Use:   "describe",
			Short: "shows all properties from a particular package from the last search result",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
		&cobra.Command{
			Use:   "tree",
			Short: "shows all dependencies for a particular package.",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
	)
}

func addOtherCommands(cmds []*cobra.Command) []*cobra.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		&cobra.Command{
			Use:   "config",
			Short: "configuration settings",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
		&cobra.Command{
			Use:   "gui",
			Short: "opens the graphical user-interface",
			Long:  headerMessage,
			Run: func(cmd *cobra.Command, args []string) {
				printDefault()
			},
		},
	)
}

func addInfoCommand(cmds []*cobra.Command) []*cobra.Command {
	return append(cmds, &cobra.Command{
		Use:   "info",
		Short: "prints ppm help message",
		Long:  headerMessage,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.Help()
		},
	})
}

var versionFlag bool

var rootCmd = &cobra.Command{
	Use:   "ppm",
	Short: "Prints hints on how to replace PPM with the state tool",
	Long:  `Shim for the deprecated Perl Package Manager, explaining how to replace it with the state tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			printVersion()
			return
		}
		printDefault()
	},
}

func main() {
	rootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "prints version information")
	var commands []*cobra.Command
	commands = addPackagesCommands(commands)
	commands = addRepositoryCommands(commands)
	commands = addProjectCommands(commands)
	commands = addVersionCommand(commands)
	commands = addInfoCommand(commands)
	commands = addOtherCommands(commands)

	// replace the --help messages with the actual commands
	for _, c := range commands {
		c.SetHelpFunc(func(c *cobra.Command, args []string) {
			c.Run(c, args)
		})
	}
	rootCmd.AddCommand(commands...)
	rootCmd.Execute()
}
