package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/ppm"
)

func printSuggestion(out output.Outputer, ppmIntent, newCommand, docLink string) error {
	out.Print(locale.Tr("ppm_print_suggestion", ppmIntent, newCommand, docLink))
	return nil
}

func printDefault(out output.Outputer) error {
	out.Print(strings.TrimSpace(locale.T("ppm_header_message")))
	return nil
}

func printMain(out output.Outputer) error {
	out.Print(locale.T("ppm_print_main"))
	return nil
}

func newPpmCommand(prime *primer.Values) *captain.Command {
	shim := ppm.NewShim(prime.Output())
	rootCmd := captain.NewHiddenShimCommand(
		"_ppm",
		nil, nil,
		func(_ *captain.Command, args []string) error {
			for _, arg := range args {
				if arg == "--version" {
					return printDefault(prime.Output())
				}
			}
			return shim.RunPPM(args...)
		},
	)

	var children []*captain.Command
	children = addPackagesCommands(prime.Output(), children)
	children = addRepositoryCommands(prime.Output(), children)
	children = addProjectCommands(prime.Output(), children)
	children = addVersionCommand(prime.Output(), children)
	children = addInfoCommand(prime.Output(), children)
	children = addOtherCommands(prime.Output(), children)

	rootCmd.AddChildren(children...)
	return rootCmd
}

func addPackagesCommands(out output.Outputer, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(out)
	return append(cmds,
		captain.NewShimCommand(
			"install",
			"installs new packages",
			func(_ *captain.Command, args []string) error {
				return shim.RunInstall(args...)
			},
		),
		captain.NewShimCommand(
			"upgrade",
			"upgrades installed packages",
			func(_ *captain.Command, args []string) error {
				return shim.RunUpgrade(args...)
			},
		),
		captain.NewShimCommand(
			"remove",
			"removes installed packages",
			func(_ *captain.Command, args []string) error {
				return shim.RunRemove(args...)
			},
		),
	)
}

func addVersionCommand(out output.Outputer, cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		captain.NewShimCommand(
			"version",
			"print version info",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
	)
}

func addProjectCommands(out output.Outputer, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(out)
	return append(cmds,
		captain.NewShimCommand(
			"area",
			"organizes packages in different areas",
			func(_ *captain.Command, _ []string) error {
				out.Print(locale.Tr("ppm_print_redundant", "state packages"))
				return nil
			},
		),
		captain.NewShimCommand(
			"list",
			"lists installed packages",
			func(_ *captain.Command, args []string) error {
				return shim.RunList(args...)
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"files",
			"lists the full path name of the files belonging to the given package, one line per file.",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"verify",
			"checks that the installed files are present and unmodified.",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
	)
}

func addRepositoryCommands(out output.Outputer, cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		// Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"repo",
			"manages package repositories",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
		captain.NewShimCommand(
			"search",
			"searches for packages in all enabled repositories",
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(out, locale.T("ppm_search_intent"), "state packages search", "state/packages.html")
			},
		),
		captain.NewShimCommand(
			"describe",
			"shows all properties from a particular package from the last search result",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
		captain.NewShimCommand(
			"tree",
			"shows all dependencies for a particular package.",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
	)
}

func addOtherCommands(out output.Outputer, cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		captain.NewShimCommand(
			"config",
			"configuration settings",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
		captain.NewShimCommand(
			"gui",
			"opens the graphical user-interface",
			func(_ *captain.Command, _ []string) error {
				return printDefault(out)
			},
		),
	)
}

func addInfoCommand(out output.Outputer, cmds []*captain.Command) []*captain.Command {
	return append(cmds, captain.NewShimCommand(
		"info",
		"prints ppm help message",
		func(_ *captain.Command, _ []string) error {
			return printMain(out)
		},
	))
}
