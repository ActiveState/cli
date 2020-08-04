package cmdtree

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func printSuggestion(prime *primer.Values, ppmIntent, newCommand, docLink string) error {
	prime.Output().Print(locale.Tr("ppm_print_suggestion", ppmIntent, newCommand, docLink))
	return nil
}

func printDefault(prime *primer.Values) error {
	prime.Output().Print(strings.TrimSpace(locale.T("ppm_header_message")))
	return nil
}

func printMain(prime *primer.Values) error {
	prime.Output().Print(locale.T("ppm_print_main"))
	return nil
}

func newPpmCommand(prime *primer.Values) *captain.Command {
	rootCmd := captain.NewHiddenShimCommand(
		"_ppm",
		nil, nil,
		func(_ *captain.Command, args []string) error {
			for _, arg := range args {
				if arg == "--version" {
					return printDefault(prime)
				}
			}
			return shim(prime, "ppm", "packages", "ppm_print_forward", args...)
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
	return append(cmds,
		captain.NewShimCommand(
			"install",
			"installs new packages",
			func(_ *captain.Command, args []string) error {
				return shim(prime, "install", "packages add", "ppm_print_forward_failure", args...)
			},
		),
		captain.NewShimCommand(
			"upgrade",
			"upgrades installed packages",
			func(_ *captain.Command, args []string) error {
				return shim(prime, "upgrade", "packages update", "ppm_print_forward_failure", args...)
			},
		),
		captain.NewShimCommand(
			"remove",
			"removes installed packages",
			func(_ *captain.Command, args []string) error {
				return shim(prime, "remove", "packages remove", "ppm_print_forward_failure", args...)
			},
		),
	)
}

func addVersionCommand(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		captain.NewShimCommand(
			"version",
			"print version info",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
	)
}

func addProjectCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		captain.NewShimCommand(
			"area",
			"organizes packages in different areas",
			func(_ *captain.Command, _ []string) error {
				prime.Output().Print(locale.Tr("ppm_print_redundant", "state packages"))
				return nil
			},
		),
		captain.NewShimCommand(
			"list",
			"lists installed packages",
			func(_ *captain.Command, args []string) error {
				return shim(prime, "list", "packages", "ppm_print_forward", args...)
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"files",
			"lists the full path name of the files belonging to the given package, one line per file.",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
		//	Long:  strings.TrimSpace(locale.T("ppm_header_message")),
		captain.NewShimCommand(
			"verify",
			"checks that the installed files are present and unmodified.",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
	)
}

func addRepositoryCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
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
				return printDefault(prime)
			},
		),
		captain.NewShimCommand(
			"search",
			"searches for packages in all enabled repositories",
			func(_ *captain.Command, _ []string) error {
				return printSuggestion(prime, locale.T("ppm_search_intent"), "state packages search", "state/packages.html")
			},
		),
		captain.NewShimCommand(
			"describe",
			"shows all properties from a particular package from the last search result",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
		captain.NewShimCommand(
			"tree",
			"shows all dependencies for a particular package.",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
	)
}

func addOtherCommands(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	return append(cmds,
		// The repo sub-commands in ppm configure alternative package
		// directories. At this point, this is an unsupported functionality, as
		// you cannot host a private platform yet.
		// So, I am just printing the default message.
		captain.NewShimCommand(
			"config",
			"configuration settings",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
		captain.NewShimCommand(
			"gui",
			"opens the graphical user-interface",
			func(_ *captain.Command, _ []string) error {
				return printDefault(prime)
			},
		),
	)
}

func addInfoCommand(prime *primer.Values, cmds []*captain.Command) []*captain.Command {
	return append(cmds, captain.NewShimCommand(
		"info",
		"prints ppm help message",
		func(_ *captain.Command, _ []string) error {
			return printMain(prime)
		},
	))
}

func shim(prime *primer.Values, intercepted, replaced, localeID string, args ...string) error {
	pj, fail := projectfile.GetSafe()
	if fail != nil && !fail.Type.Matches(projectfile.FailNoProject) {
		return locale.WrapError(fail.ToError(), "err_ppm_get_projectfile", "Encountered unexpected error loading projectfile")
	}
	if pj == nil {
		// TODO: Replace this function call when conversion flow is complete
		return tutorial()
	}

	commands := strings.Split(replaced, " ")
	replacedArgs := args
	if len(commands) > 1 {
		replaced = commands[0]
		replacedArgs = commands[1:]
		replacedArgs = append(replacedArgs, args...)
	}

	forwarded := []string{"state", replaced}
	forwarded = append(forwarded, replacedArgs...)
	prime.Output().Print(locale.Tr(localeID, strings.Join(forwarded, " "), intercepted))

	return invoke(replaced, replacedArgs...)
}

func invoke(command string, args ...string) error {
	executable, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_invoke_executable", "Could not find State Tool executable")
	}

	commandArgs := []string{command}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command(executable, commandArgs...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdout
	return cmd.Run()
}

func tutorial() error {
	// Placeholder until conversion flow is complete
	return nil
}
