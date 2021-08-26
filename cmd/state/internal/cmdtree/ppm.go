package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/ppm"
)

func newPpmCommand(prime *primer.Values) *captain.Command {
	shim := ppm.NewShim(prime)
	conversion := ppm.NewConversionFlow(prime)
	rootCmd := captain.NewHiddenShimCommand(
		"_ppm",
		nil, nil,
		func(_ *captain.Command, args []string) error {
			for _, arg := range args {
				if arg == "--version" {
					return shim.PrintDefault()
				}
			}
			converted, err := conversion.StartIfNecessary()
			if err != nil {
				return err
			}
			return shim.RunPPM(converted, args...)
		},
	)

	var children []*captain.Command
	children = addPackagesCommands(prime, children)
	children = addRepositoryCommands(prime, children)
	children = addProjectCommands(prime, children)
	children = addVersionCommand(prime, children)
	children = addInfoCommand(prime, children)
	children = addOtherCommands(prime, children)

	rootCmd.AddChildren(children...)
	return rootCmd
}

func addPackagesCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
	conversion := ppm.NewConversionFlow(prime)
	return append(cmds,
		captain.NewShimCommand(
			"install",
			"installs new packages",
			func(_ *captain.Command, args []string) error {
				converted, err := conversion.StartIfNecessary()
				if err != nil {
					return err
				}

				return shim.RunInstall(converted, args...)
			},
		),
		captain.NewShimCommand(
			"upgrade",
			"upgrades installed packages",
			func(_ *captain.Command, args []string) error {
				converted, err := conversion.StartIfNecessary()
				if err != nil {
					return err
				}

				return shim.RunUpgrade(converted, args...)
			},
		),
		captain.NewShimCommand(
			"remove",
			"removes installed packages",
			func(_ *captain.Command, args []string) error {
				converted, err := conversion.StartIfNecessary()
				if err != nil {
					return err
				}

				return shim.RunRemove(converted, args...)
			},
		),
	)
}

func addVersionCommand(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
	return append(cmds,
		captain.NewShimCommand(
			"version",
			"print version info",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
	)
}

func addProjectCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
	conversion := ppm.NewConversionFlow(prime)
	return append(cmds,
		captain.NewShimCommand(
			"area",
			"organizes packages in different areas",
			func(_ *captain.Command, _ []string) error {
				_, err := conversion.StartIfNecessary()
				if err != nil {
					return err
				}

				prime.Output().Print(locale.Tr("ppm_print_area_redundant", "state packages"))
				return nil
			},
		),
		captain.NewShimCommand(
			"list",
			"lists installed packages",
			func(_ *captain.Command, args []string) error {
				converted, err := conversion.StartIfNecessary()
				if err != nil {
					return err
				}

				return shim.RunList(converted, args...)
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"files",
			"lists the full path name of the files belonging to the given package, one line per file.",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"verify",
			"checks that the installed files are present and unmodified.",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
	)
}

func addRepositoryCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
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
				return shim.PrintDefault()
			},
		),
		captain.NewShimCommand(
			"search",
			"searches for packages in all enabled repositories",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintSuggestion(locale.T("ppm_search_intent"), "state packages search", "state/packages.html")
			},
		),
		captain.NewShimCommand(
			"describe",
			"shows all properties from a particular package from the last search result",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
		captain.NewShimCommand(
			"tree",
			"shows all dependencies for a particular package.",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
	)
}

func addOtherCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		captain.NewShimCommand(
			"config",
			"configuration settings",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
		captain.NewShimCommand(
			"gui",
			"opens the graphical user-interface",
			func(_ *captain.Command, _ []string) error {
				return shim.PrintDefault()
			},
		),
	)
}

func addInfoCommand(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
	return append(cmds, captain.NewShimCommand(
		"info",
		"prints ppm help message",
		func(_ *captain.Command, _ []string) error {
			return shim.PrintMain()
		},
	))
}
