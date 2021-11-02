package exec

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/shirou/gopsutil/process"
)

type Exec struct {
	subshell  subshell.SubShell
	proj      *project.Project
	auth      *authentication.Auth
	out       output.Outputer
	cfg       projectfile.ConfigGetter
	analytics analytics.Dispatcher
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Subsheller
	primer.Projecter
	primer.Configurer
	primer.Analyticer
}

type Params struct {
	Path string
}

func New(prime primeable) *Exec {
	return &Exec{
		prime.Subshell(),
		prime.Project(),
		prime.Auth(),
		prime.Output(),
		prime.Config(),
		prime.Analytics(),
	}
}

func NewParams() *Params {
	return &Params{}
}

func (s *Exec) Run(params *Params, args ...string) error {
	var projectDir string
	var rtTarget setup.Targeter

	if len(args) == 0 {
		return nil
	}

	trigger := runtime.NewExecTrigger(args[0])

	// Detect target and project dir
	// If the path passed resolves to a runtime dir (ie. has a runtime marker) then the project is not used
	if params.Path != "" && runtime.IsRuntimeDir(params.Path) {
		projectDir = projectFromRuntimeDir(s.cfg, params.Path)
		proj, err := project.FromPath(projectDir)
		if err != nil {
			logging.Error("Could not get project dir from path: %s", errs.JoinMessage(err))
			// We do not know if the project is headless at this point so we default to true
			// as there is no head
			rtTarget = runtime.NewCustomTarget("", "", "", params.Path, trigger, true)
		} else {
			rtTarget = runtime.NewProjectTarget(proj, storage.CachePath(), nil, trigger)
		}
	} else {
		proj := s.proj
		if params.Path != "" {
			var err error
			proj, err = project.FromPath(params.Path)
			if err != nil {
				return locale.WrapInputError(err, "exec_no_project_at_path", "Could not find project file at {{.V0}}", params.Path)
			}
		}
		if s.proj == nil {
			return locale.NewError("exec_no_project_found", "Could not find a project.  You need to be in a project directory or specify a global default project via `state activate --default`")
		}
		projectDir = filepath.Dir(proj.Source().Path())
		rtTarget = runtime.NewProjectTarget(proj, storage.CachePath(), nil, trigger)
	}

	rt, err := runtime.New(rtTarget, s.analytics)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}
		eh, err := runbits.DefaultRuntimeEventHandler(s.out)
		if err != nil {
			return locale.WrapError(err, "err_initialize_runtime_event_handler")
		}
		if err := rt.Update(s.auth, eh); err != nil {
			return locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
	}
	venv := virtualenvironment.New(rt)

	env, err := venv.GetEnv(true, false, projectDir)
	if err != nil {
		return locale.WrapError(err, "err_exec_env", "Could not retrieve environment information for your runtime")
	}
	logging.Debug("Trying to exec %s on PATH=%s", args[0], env["PATH"])

	if err := handleRecursion(env, args); err != nil {
		return errs.Wrap(err, "Could not handle recursion")
	}

	exeTarget := args[0]
	if !fileutils.TargetExists(exeTarget) {
		rtExePaths, err := rt.ExecutablePaths()
		if err != nil {
			return errs.Wrap(err, "Could not detect runtime executable paths")
		}
		RTPATH := strings.Join(rtExePaths, string(os.PathListSeparator)) + string(os.PathListSeparator)

		// Report recursive execution of executor: The path for the executable should be different from the default bin dir
		exesOnPath := exeutils.FilterExesOnPATH(args[0], RTPATH, func(exe string) bool {
			v, err := executor.IsExecutor(exe)
			if err != nil {
				logging.Error("Could not find out if executable is an executor: %s", errs.JoinMessage(err))
				return true // This usually means there's a permission issue, which means we likely don't own it
			}
			return !v
		})

		if len(exesOnPath) > 0 {
			exeTarget = exesOnPath[0]
		}
	}

	// Guard against invoking the executor from PATH (ie. by name alone)
	if os.Getenv(constants.ExecRecursionAllowEnvVarName) != "true" && filepath.Base(exeTarget) == exeTarget { // not a full path
		exe := exeutils.FindExeInside(exeTarget, env["PATH"])
		if exe != exeTarget { // Found the exe name on our PATH
			isExec, err := executor.IsExecutor(exe)
			if err != nil {
				logging.Error("Could not find out if executable is an executor: %s", errs.JoinMessage(err))
			} else if isExec {
				// If the exe we resolve to is an executor then we have ourselves a recursive loop
				return locale.NewError("err_exec_recursion", "", constants.ForumsURL, constants.ExecRecursionAllowEnvVarName)
			}
		}
	}

	s.subshell.SetEnv(env)

	lang := language.Bash
	scriptArgs := fmt.Sprintf(`%s "$@"`, exeTarget)
	if strings.Contains(s.subshell.Binary(), "cmd") {
		lang = language.Batch
		scriptArgs = fmt.Sprintf("@ECHO OFF\n%s %%*", exeTarget)
	}

	sf, err := scriptfile.New(lang, "state-exec", scriptArgs)
	if err != nil {
		return locale.WrapError(err, "err_exec_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename(), args[1:]...)
}

func projectFromRuntimeDir(cfg projectfile.ConfigGetter, runtimeDir string) string {
	projects := projectfile.GetProjectMapping(cfg)
	for _, paths := range projects {
		for _, p := range paths {
			targetBase := hash.ShortHash(p)
			if filepath.Base(runtimeDir) == targetBase {
				return p
			}
		}
	}

	return ""
}

func handleRecursion(env map[string]string, args []string) error {
	recursionReadable := []string{}
	recursionReadableFull := os.Getenv(constants.ExecRecursionEnvVarName)
	if recursionReadableFull == "" {
		recursionReadable = append(recursionReadable, getParentProcessArgs())
	} else {
		recursionReadable = strings.Split(recursionReadableFull, "\n")
	}
	recursionReadable = append(recursionReadable, filepath.Base(os.Args[0])+" "+strings.Join(os.Args[1:], " "))
	var recursionLvl int64
	lastLvl, err := strconv.ParseInt(os.Getenv(constants.ExecRecursionLevelEnvVarName), 10, 32)
	if err == nil {
		recursionLvl = lastLvl + 1
	}
	maxLevel, err := strconv.ParseInt(os.Getenv(constants.ExecRecursionMaxLevelEnvVarName), 10, 32)
	if err == nil || maxLevel == 0 {
		maxLevel = 10
	}
	if recursionLvl == 2 || recursionLvl == 10 || recursionLvl == 50 {
		logging.Error("executor recursion detected: parent %s (%d): %s (lvl=%d)", getParentProcessArgs(), os.Getppid(), strings.Join(args, " "), recursionLvl)
	}
	if recursionLvl >= maxLevel {
		return locale.NewError("err_recursion_limit", "", strings.Join(recursionReadable, "\n"), constants.ExecRecursionMaxLevelEnvVarName)
	}

	env[constants.ExecRecursionLevelEnvVarName] = fmt.Sprintf("%d", recursionLvl)
	env[constants.ExecRecursionMaxLevelEnvVarName] = fmt.Sprintf("%d", maxLevel)
	env[constants.ExecRecursionEnvVarName] = strings.Join(recursionReadable, "\n")
	return nil
}

func getParentProcessArgs() string {
	p, err := process.NewProcess(int32(os.Getppid()))
	if err != nil {
		logging.Debug("Could not find parent process of executor: %v", err)
		return "unknown"
	}

	args, err := p.CmdlineSlice()
	if err != nil {
		logging.Debug("Could not retrieve command line arguments of executor's calling process: %v", err)
		return "unknown"
	}

	return strings.Join(args, " ")
}
