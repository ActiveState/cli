package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/ppm"
)

<<<<<<< HEAD
func newPpmCommand(prime *primer.Values) *captain.Command {
	shim := ppm.NewShim(prime)
=======
func printSuggestion(ppmIntent, newCommand, docLink string) error {
	fmt.Println(locale.Tr("ppm_print_suggestion", ppmIntent, newCommand, docLink))
	return nil
}

func printDefault() error {
	fmt.Println(strings.TrimSpace(locale.T("ppm_header_message")))
	return nil
}

func printMain() error {
	fmt.Println(locale.T("ppm_print_main"))
	return nil
}

func newPpmCommand(prime *primer.Values) *captain.Command {
	runner := ppm.New(prime)
>>>>>>> add conversion flow
	rootCmd := captain.NewHiddenShimCommand(
		"_ppm",
		nil, nil,
		func(_ *captain.Command, args []string) error {
			for _, arg := range args {
				if arg == "--version" {
					return shim.PrintDefault()
				}
			}
<<<<<<< HEAD
			return shim.RunPPM(args...)
=======
			err := runner.StartConversionFlowIfNecessary()
			if err != nil {
				return err
			}
			return printMain()
>>>>>>> add conversion flow
		},
	)

	var children []*captain.Command
<<<<<<< HEAD
	children = addPackagesCommands(prime, children)
	children = addRepositoryCommands(prime, children)
	children = addProjectCommands(prime, children)
	children = addVersionCommand(prime, children)
	children = addInfoCommand(prime, children)
	children = addOtherCommands(prime, children)
=======
	children = addPackagesCommands(runner, children)
	children = addRepositoryCommands(children)
	children = addProjectCommands(runner, children)
	children = addVersionCommand(children)
	children = addInfoCommand(children)
	children = addOtherCommands(children)
>>>>>>> add conversion flow

	rootCmd.AddChildren(children...)
	return rootCmd
}

<<<<<<< HEAD
func addPackagesCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
=======
func addPackagesCommands(runner *ppm.Ppm, cmds []*captain.Command) []*captain.Command {
>>>>>>> add conversion flow
	return append(cmds,
		captain.NewShimCommand(
			"install",
			"installs new packages",
<<<<<<< HEAD
			func(_ *captain.Command, args []string) error {
				return shim.RunInstall(args...)
=======
			func(_ *captain.Command, _ []string) error {
				err := runner.StartConversionFlowIfNecessary()
				if err != nil {
					return err
				}

				return printSuggestion(locale.T("ppm_install_intent"), "state packages add", "state/packages.html")
>>>>>>> add conversion flow
			},
		),
		captain.NewShimCommand(
			"upgrade",
			"upgrades installed packages",
<<<<<<< HEAD
			func(_ *captain.Command, args []string) error {
				return shim.RunUpgrade(args...)
=======
			func(_ *captain.Command, _ []string) error {
				err := runner.StartConversionFlowIfNecessary()
				if err != nil {
					return err
				}

				return printSuggestion(locale.T("ppm_upgrade_intent"), "state packages update", "state/packages.html")
>>>>>>> add conversion flow
			},
		),
		captain.NewShimCommand(
			"remove",
			"removes installed packages",
<<<<<<< HEAD
			func(_ *captain.Command, args []string) error {
				return shim.RunRemove(args...)
=======
			func(_ *captain.Command, _ []string) error {
				err := runner.StartConversionFlowIfNecessary()
				if err != nil {
					return err
				}

				return printSuggestion(locale.T("ppm_remove_intent"), "state packages remove", "state/packages.html")
>>>>>>> add conversion flow
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

<<<<<<< HEAD
func addProjectCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	shim := ppm.NewShim(prime)
=======
func addProjectCommands(runner *ppm.Ppm, cmds []*captain.Command) []*captain.Command {
>>>>>>> add conversion flow
	return append(cmds,
		captain.NewShimCommand(
			"area",
			"organizes packages in different areas",
			func(_ *captain.Command, _ []string) error {
<<<<<<< HEAD
				prime.Output().Print(locale.Tr("ppm_print_redundant", "state packages"))
=======
				err := runner.StartConversionFlowIfNecessary()
				if err != nil {
					return err
				}

				fmt.Println(locale.T("ppm_area_message"))
>>>>>>> add conversion flow
				return nil
			},
		),
		captain.NewShimCommand(
			"list",
			"lists installed packages",
<<<<<<< HEAD
			func(_ *captain.Command, args []string) error {
				return shim.RunList(args...)
=======
			func(_ *captain.Command, _ []string) error {
				err := runner.StartConversionFlowIfNecessary()
				if err != nil {
					return err
				}

				return printSuggestion(locale.T("ppm_list_intent"), "state packages", "state/packages.html")
>>>>>>> add conversion flow
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
