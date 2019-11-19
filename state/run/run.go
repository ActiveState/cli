package run

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
)

// Command holds the definition for "state run".
var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:               "run",
		Description:        "run_description",
		Run:                Execute,
		DisableFlagParsing: true,

		Arguments: []*commands.Argument{
			&commands.Argument{
				Name:        "arg_state_run_name",
				Description: "arg_state_run_name_description",
				Variable:    &Args.Name,
			},
		},
	}
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// Execute the run command.
func Execute(cmd *cobra.Command, allArgs []string) {
	checker.RunCommitsBehindNotifier()

	logging.Debug("Execute")

	if Args.Name == "" || strings.HasPrefix(Args.Name, "-") {
		failures.Handle(failures.FailUserInput.New("error_state_run_undefined_name"), "")
		return
	}

	scriptArgs := allArgs[1:]

	// Determine which project script to run based on the given script name.
	script := project.Get().ScriptByName(Args.Name)
	if script == nil {
		print.Error(locale.T("error_state_run_unknown_name", map[string]string{"Name": Args.Name}))
		return
	}

	subs, fail := subshell.Get()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_run_no_shell"))
		return
	}

	lang := script.Language()
	if lang == language.Unknown {
		lang = language.MakeByShell(subs.Shell())
	}

	langExec := lang.Executable()
	if script.Standalone() && !langExec.Builtin() {
		print.Error(locale.T("error_state_run_standalone_conflict"))
		return
	}

	path := os.Getenv("PATH")

	// Activate the state if needed.
	if !script.Standalone() && !subshell.IsActivated() {
		print.Info(locale.T("info_state_run_activating_state"))
		venv := virtualenvironment.Init()
		venv.OnDownloadArtifacts(func() { print.Line(locale.T("downloading_artifacts")) })
		venv.OnInstallArtifacts(func() { print.Line(locale.T("installing_artifacts")) })

		if fail := venv.Activate(); fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			failures.Handle(fail, locale.T("error_state_run_activate"))
			return
		}

		subs.SetEnv(venv.GetEnvSlice(true))
		path = venv.GetEnv()["PATH"]
	}

	if !langExec.Builtin() && !pathProvidesExec(configCachePath(), langExec.Name(), path) {
		print.Error(locale.T("error_state_run_unknown_exec"))
		return
	}

	// Run the script.
	scriptBlock := project.Expand(script.Value())
	sf, fail := scriptfile.New(lang, script.Name(), scriptBlock)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_run_setup_scriptfile"))
		return
	}
	defer sf.Clean()

	print.Info(locale.Tr("info_state_run_running", script.Name(), script.Source().Path()))
	// ignore code for now, passing via failure
	_, err := subs.Run(sf.Filename(), scriptArgs...)
	if err != nil {
		failures.Handle(err, locale.T("error_state_run_error"))
	}
}

func configCachePath() string {
	if runtime.GOOS == "darwin" { // runtime loading is not yet supported in darwin systems
		return "" // empty string value will skip path filtering in subsequent logic
	}
	return config.CachePath()
}

func pathProvidesExec(filterByPath, exec, path string) bool {
	paths := splitPath(path)
	if filterByPath != "" {
		paths = filterPrefixed(filterByPath, paths)
	}
	paths = applySuffix(exec, paths)

	for _, p := range paths {
		if isExecutableFile(p) {
			return true
		}
	}
	return false
}

func splitPath(path string) []string {
	return strings.Split(path, string(os.PathListSeparator))
}

func filterPrefixed(prefix string, paths []string) []string {
	var ps []string
	for _, p := range paths {
		if strings.HasPrefix(p, prefix) {
			ps = append(ps, p)
		}
	}
	return ps
}

func applySuffix(suffix string, paths []string) []string {
	for i, v := range paths {
		paths[i] = filepath.Join(v, suffix)
	}
	return paths
}

func isExecutableFile(name string) bool {
	f, err := os.Stat(name)
	if err != nil { // unlikely unless file does not exist
		return false
	}

	if runtime.GOOS == "windows" {
		return f.Mode()&0400 != 0
	}

	return f.Mode()&0110 != 0
}
