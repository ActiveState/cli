package runtime_runbit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime/progress"
	"github.com/ActiveState/cli/internal/runbits/runtime/target"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/go-openapi/strfmt"
)

func init() {
	configMediator.RegisterOption(constants.AsyncRuntimeConfig, configMediator.Bool, false)
}

type Opts struct {
	PrintHeaders bool
	TargetDir    string

	// Note CommitID and Commit are mutually exclusive. If Commit is provided then CommitID is disregarded.
	CommitID strfmt.UUID
	Commit   *bpModel.Commit
}

type SetOpt func(*Opts)

func WithPrintHeaders(printHeaders bool) SetOpt {
	return func(opts *Opts) {
		opts.PrintHeaders = printHeaders
	}
}

func WithTargetDir(targetDir string) SetOpt {
	return func(opts *Opts) {
		opts.TargetDir = targetDir
	}
}

func WithCommit(commit *bpModel.Commit) SetOpt {
	return func(opts *Opts) {
		opts.Commit = commit
	}
}

func WithCommitID(commitID strfmt.UUID) SetOpt {
	return func(opts *Opts) {
		opts.CommitID = commitID
	}
}

var overrideAsyncTriggers = map[target.Trigger]bool{
	target.TriggerRefresh:  true,
	target.TriggerExec:     true,
	target.TriggerActivate: true,
	target.TriggerShell:    true,
	target.TriggerScript:   true,
	target.TriggerDeploy:   true,
	target.TriggerUse:      true,
}

type solvePrimer interface {
	primer.Projecter
	primer.Auther
	primer.Outputer
}

func Solve(
	prime solvePrimer,
	overrideCommitID *strfmt.UUID,
) (_ *bpModel.Commit, rerr error) {
	defer rationalizeSolveError(prime, &rerr)

	proj := prime.Project()

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	var err error
	var commitID strfmt.UUID
	if overrideCommitID != nil {
		commitID = *overrideCommitID
	} else {
		commitID, err = localcommit.Get(proj.Dir())
		if err != nil {
			return nil, errs.Wrap(err, "Failed to get local commit")
		}
	}

	solveSpinner := output.StartSpinner(prime.Output(), locale.T("progress_solve"), constants.TerminalAnimationInterval)

	bpm := bpModel.NewBuildPlannerModel(prime.Auth())
	commit, err := bpm.FetchCommit(commitID, proj.Owner(), proj.Name(), nil)
	if err != nil {
		solveSpinner.Stop(locale.T("progress_fail"))
		return nil, errs.Wrap(err, "Failed to fetch build result")
	}

	solveSpinner.Stop(locale.T("progress_success"))

	return commit, nil
}

type updatePrimer interface {
	primer.Projecter
	primer.Auther
	primer.Outputer
	primer.Configurer
}

func Update(
	prime updatePrimer,
	trigger target.Trigger,
	setOpts ...SetOpt,
) (_ *runtime.Runtime, rerr error) {
	defer rationalizeUpdateError(prime, &rerr)

	opts := &Opts{}
	for _, setOpt := range setOpts {
		setOpt(opts)
	}

	proj := prime.Project()

	if proj == nil {
		return nil, rationalize.ErrNoProject
	}

	targetDir := opts.TargetDir
	if targetDir == "" {
		targetDir = targetDirFromProject(proj)
	}

	rt, err := runtime.New(targetDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not initialize runtime")
	}

	optinBuildscripts := prime.Config().GetBool(constants.OptinBuildscriptsConfig)

	commitID := opts.CommitID
	if commitID == "" {
		commitID, err = localcommit.Get(proj.Dir())
		if err != nil {
			return nil, errs.Wrap(err, "Failed to get local commit")
		}
	}

	commitHash := string(commitID)
	if optinBuildscripts {
		bs, err := fileutils.ReadFile(filepath.Join(proj.Dir(), constants.BuildScriptFileName))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, ErrBuildscriptNotExist
			}
			return nil, errs.Wrap(err, "Unknown failure while reading buildscript file")
		}
		commitHash += string(bs)
	}

	rtHash := hash.ShortHash(strings.Join([]string{proj.NamespaceString(), proj.Dir(), commitHash}, ""))
	if optinBuildscripts && rt.Hash() != "" && rt.Hash() != rtHash {
		return nil, ErrBuildScriptNeedsCommit
	}

	if opts.PrintHeaders {
		if !rt.HasCache() {
			prime.Output().Notice(output.Title(locale.T("install_runtime")))
		} else {
			prime.Output().Notice(output.Title(locale.T("update_runtime")))
		}
	}

	if rt.Hash() == rtHash {
		prime.Output().Notice(locale.T("pkg_already_uptodate"))
		return rt, nil
	}

	commit := opts.Commit
	if commit == nil {
		commit, err = Solve(prime, &commitID)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to solve runtime")
		}
	}

	// Async runtimes should still do everything up to the actual update itself, because we still want to raise
	// any errors regarding solves, buildscripts, etc.
	if prime.Config().GetBool(constants.AsyncRuntimeConfig) && !overrideAsyncTriggers[trigger] {
		logging.Debug("Skipping runtime update due to async runtime")
		return rt, nil
	}

	pg := progress.NewRuntimeProgressIndicator(prime.Output())
	defer rtutils.Closer(pg.Close, &rerr)
	if err := rt.Update(commit.BuildPlan(), rtHash,
		runtime.WithAnnotations(proj.Owner(), proj.Name(), commitID),
		runtime.WithEventHandlers(pg.Handle),
		runtime.WithPreferredLibcVersion(prime.Config().GetString(constants.PreferredGlibcVersionConfig)),
	); err != nil {
		return nil, locale.WrapError(err, "err_packages_update_runtime_install")
	}

	return rt, nil
}

func targetDirFromProject(proj *project.Project) string {
	if cache := proj.Cache(); cache != "" {
		return cache
	}

	resolvedDir, err := fileutils.ResolveUniquePath(proj.Dir())
	if err != nil {
		multilog.Error("Could not resolve unique path for projectDir: %s, error: %s", proj.Dir(), err.Error())
		resolvedDir = proj.Dir()
	}

	return filepath.Join(storage.CachePath(), hash.ShortHash(resolvedDir))
}
