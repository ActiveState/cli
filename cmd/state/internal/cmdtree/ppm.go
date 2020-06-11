package cmdtree

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
)

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

func newPpmCommand() *captain.Command {
	var versionFlagSet bool

	rootCmd := captain.NewHiddenShimCommand(
		"_ppm",
		[]*captain.Flag{
			{
				Name:        "version",
				Description: "prints version information",
				Persist:     true,
				Value:       &versionFlagSet,
			},
		},
		nil,
		func(_ *captain.Command, _ []string) error {
			if versionFlagSet {
				return printDefault()
			}
			return printMain()
		},
	)

	var children []*captain.Command
	children = addPackagesCommands(children)
	children = addRepositoryCommands(children)
	children = addProjectCommands(children)
	children = addVersionCommand(children)
	children = addInfoCommand(children)
	children = addOtherCommands(children)

	rootCmd.AddChildren(children...)
	return rootCmd
}

func addPackagesCommands(cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		captain.NewCommand(
			"install",
			"installs new packages",
			nil,
			nil,
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(locale.T("ppm_install_intent"), "state packages add", "state/packages.html")
			},
		),
		captain.NewCommand(
			"upgrade",
			"upgrades installed packages",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(locale.T("ppm_upgrade_intent"), "state packages update", "state/packages.html")
			},
		),
		captain.NewCommand(
			"remove",
			"removes installed packages",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(locale.T("ppm_remove_intent"), "state packages remove", "state/packages.html")
			},
		),
	)
}

func addVersionCommand(cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		captain.NewCommand(
			"version",
			"print version info",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
	)
}

func addProjectCommands(cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		captain.NewCommand(
			"area",
			"organizes packages in different areas",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				fmt.Println(locale.T("ppm_area_message"))
				return nil
			},
		),
		captain.NewCommand(
			"list",
			"lists installed packages",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(locale.T("ppm_list_intent"), "state packages", "state/packages.html")
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewCommand(
			"files",
			"lists the full path name of the files belonging to the given package, one line per file.",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewCommand(
			"verify",
			"checks that the installed files are present and unmodified.",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
	)
}

func addRepositoryCommands(cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		// Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewCommand(
			"repo",
			"manages package repositories",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
		captain.NewCommand(
			"search",
			"searches for packages in all enabled repositories",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(locale.T("ppm_search_intent"), "state packages search", "state/packages.html")
			},
		),
		captain.NewCommand(
			"describe",
			"shows all properties from a particular package from the last search result",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
		captain.NewCommand(
			"tree",
			"shows all dependencies for a particular package.",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
	)
}

func addOtherCommands(cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		captain.NewCommand(
			"config",
			"configuration settings",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
		captain.NewCommand(
			"gui",
			"opens the graphical user-interface",
			nil, nil,
			func(_ *captain.Command, _ []string) error {
				return printDefault()
			},
		),
	)
}

func addInfoCommand(cmds []*captain.Command) []*captain.Command {
	return append(cmds, captain.NewCommand(
		"info",
		"prints ppm help message",
		nil, nil,
		func(_ *captain.Command, _ []string) error {
			return printMain()
		},
	))
}
