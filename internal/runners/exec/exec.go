package exec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/pkg/executors"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	rt "github.com/ActiveState/cli/pkg/runtime"
)

type Configurable interface {
	projectfile.ConfigGetter
	GetBool(key string) bool
}

type Exec struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	subshell  subshell.SubShell
	proj      *project.Project
	auth      *authentication.Auth
	out       output.Outputer
	cfg       Configurable
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
		prime,
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

	if len(args) == 0 {
		return nil
	}

	trigger := trigger.NewExecTrigger(args[0])

	// Detect target and project dir
	// If the path passed resolves to a runtime dir (ie. has a runtime marker) then the project is not used
	var proj *project.Project
	var err error
	if params.Path != "" && rt.IsRuntimeDir(params.Path) {
		projectDir = projectFromRuntimeDir(s.cfg, params.Path)
		proj, err = project.FromPath(projectDir)
		if err != nil {
			return locale.WrapInputError(err, "exec_no_project_at_path", "Could not find project file at {{.V0}}", projectDir)
		}
		projectNamespace = proj.NamespaceString()
	} else {
		proj = s.proj
		if params.Path != "" {
			var err error
			proj, err = project.FromPath(params.Path)
			if err != nil {
				return locale.WrapInputError(err, "exec_no_project_at_path", "Could not find project file at {{.V0}}", params.Path)
			}
		}
		if proj == nil {
			return rationalize.ErrNoProject
		}
		projectDir = filepath.Dir(proj.Source().Path())
		projectNamespace = proj.NamespaceString()
	}

	s.prime.SetProject(proj)

	s.out.Notice(locale.Tr("operating_message", projectNamespace, projectDir))

	rt, err := runtime_runbit.Update(s.prime, trigger, runtime_runbit.WithoutHeaders(), runtime_runbit.WithIgnoreAsync())
	if err != nil {
		return errs.Wrap(err, "Could not initialize runtime")
	}

	venv := virtualenvironment.New(rt)

	env, err := venv.GetEnv(true, false, projectDir, projectNamespace)
	if err != nil {
		return locale.WrapError(err, "err_exec_env", "Could not retrieve environment information for your runtime")
	}

	exeTarget := args[0]

	if err := handleRecursion(env, args); err != nil {
		return errs.Wrap(err, "Could not handle recursion")
	}

	if !fileutils.TargetExists(exeTarget) {
		// Report recursive execution of executor: The path for the executable should be different from the default bin dir
		filter := func(exe string) bool {
			v, err := executors.IsExecutor(exe)
			if err != nil {
				logging.Error("Could not find out if executable is an executor: %s", errs.JoinMessage(err))
				return true // This usually means there's a permission issue, which means we likely don't own it
			}
			return !v
		}
		exesOnPath := osutils.FilterExesOnPATH(exeTarget, env["PATH"], filter)
		if runtime.GOOS == "windows" {
			exesOnPath = append(exesOnPath, osutils.FilterExesOnPATH(exeTarget, env["Path"], filter)...)
		}

		if len(exesOnPath) > 0 {
			exeTarget = exesOnPath[0]
		} else {
			return errs.AddTips(locale.NewInputError(
				"err_exec_not_found",
				"The executable '{{.V0}}' was not found in your PATH or in your project runtime.",
				exeTarget),
				locale.Tl("err_exec_not_found_tip", "Run '[ACTIONABLE]state export env[/RESET]' to check project runtime paths"))
		}
	}

	// Guard against invoking the executor from PATH (ie. by name alone)
	if os.Getenv(constants.ExecRecursionAllowEnvVarName) != "true" && filepath.Base(exeTarget) == exeTarget { // not a full path
		exe := osutils.FindExeInside(exeTarget, env["PATH"])
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

	_, _, err = osutils.ExecuteAndPipeStd(exeTarget, args[1:], osutils.EnvMapToSlice(env))
	if eerr, ok := err.(*exec.ExitError); ok {
		return errs.Silence(errs.WrapExitCode(eerr, eerr.ExitCode()))
	}
	if err != nil {
		return errs.Wrap(err, "Could not execute command")
	}

	return nil
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
