package exec

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal-as/analytics"
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/internal-as/logging"
	"github.com/ActiveState/cli/internal-as/multilog"
	"github.com/ActiveState/cli/internal-as/output"
	"github.com/ActiveState/cli/internal-as/primer"
	"github.com/ActiveState/cli/internal-as/rtutils"
	"github.com/ActiveState/cli/internal-as/runbits"
	"github.com/ActiveState/cli/internal-as/subshell"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/executors"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/shirou/gopsutil/v3/process"
)

type Exec struct {
	subshell  subshell.SubShell
	proj      *project.Project
	auth      *authentication.Auth
	out       output.Outputer
	cfg       projectfile.ConfigGetter
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Subsheller
	primer.Projecter
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
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
		prime.SvcModel(),
	}
}

func NewParams() *Params {
	return &Params{}
}

func (s *Exec) Run(params *Params, args ...string) (rerr error) {
	var projectDir string
	var projectNamespace string
	var rtTarget setup.Targeter

	if len(args) == 0 {
		return nil
	}

	trigger := target.NewExecTrigger(args[0])

	// Detect target and project dir
	// If the path passed resolves to a runtime dir (ie. has a runtime marker) then the project is not used
	if params.Path != "" && runtime.IsRuntimeDir(params.Path) {
		projectDir = projectFromRuntimeDir(s.cfg, params.Path)
		proj, err := project.FromPath(projectDir)
		if err != nil {
			logging.Warning("Could not get project dir from path: %s", errs.JoinMessage(err))
			// We do not know if the project is headless at this point so we default to true
			// as there is no head
			rtTarget = target.NewCustomTarget("", "", "", params.Path, trigger, true)
		} else {
			rtTarget = target.NewProjectTarget(proj, storage.CachePath(), nil, trigger)
		}
		projectNamespace = proj.NamespaceString()
	} else {
		proj := s.proj
		if params.Path != "" {
			var err error
			proj, err = project.FromPath(params.Path)
			if err != nil {
				return locale.WrapInputError(err, "exec_no_project_at_path", "Could not find project file at {{.V0}}", params.Path)
			}
		}
		if proj == nil {
			return locale.NewInputError("err_no_project")
		}
		projectDir = filepath.Dir(proj.Source().Path())
		projectNamespace = proj.NamespaceString()
		rtTarget = target.NewProjectTarget(proj, storage.CachePath(), nil, trigger)
	}

	s.out.Notice(locale.Tl("operating_message", "", projectNamespace, projectDir))

	rt, err := runtime.New(rtTarget, s.analytics, s.svcModel)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}
		pg := runbits.NewRuntimeProgressIndicator(s.out)
		defer rtutils.Closer(pg.Close, &rerr)
		if err := rt.Update(s.auth, pg); err != nil {
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
		rtDirs, err := rt.ExecutableDirs()
		if err != nil {
			return errs.Wrap(err, "Could not detect runtime executable paths")
		}

		RTPATH := strings.Join(rtDirs, string(os.PathListSeparator))

		// Report recursive execution of executor: The path for the executable should be different from the default bin dir
		exesOnPath := exeutils.FilterExesOnPATH(args[0], RTPATH, func(exe string) bool {
			v, err := executors.IsExecutor(exe)
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
			isExec, err := executors.IsExecutor(exe)
			if err != nil {
				logging.Error("Could not find out if executable is an executor: %s", errs.JoinMessage(err))
			} else if isExec {
				// If the exe we resolve to is an executor then we have ourselves a recursive loop
				return locale.NewError("err_exec_recursion", "", constants.ForumsURL, constants.ExecRecursionAllowEnvVarName)
			}
		}
	}

	err = s.subshell.SetEnv(env)
	if err != nil {
		return locale.WrapError(err, "err_subshell_setenv")
	}

	lang := language.Bash
	scriptArgs := fmt.Sprintf(`%q "$@"`, exeTarget)
	if strings.Contains(s.subshell.Binary(), "cmd") {
		lang = language.Batch
		scriptArgs = fmt.Sprintf("@ECHO OFF\n%q %%*", exeTarget)
	}

	sf, err := scriptfile.New(lang, "state-exec", scriptArgs)
	if err != nil {
		return locale.WrapError(err, "err_exec_create_scriptfile", "Could not generate script")
	}
	defer sf.Clean()

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
		multilog.Error("executor recursion detected: parent %s (%d): %s (lvl=%d)", getParentProcessArgs(), os.Getppid(), strings.Join(args, " "), recursionLvl)
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
