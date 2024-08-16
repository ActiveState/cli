package checkout

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/checkout"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type Params struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Branch        string
	RuntimePath   string
	NoClone       bool
	Force         bool
}

type primeable interface {
	primer.Auther
	primer.Prompter
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
	primer.Projecter
}

type Checkout struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	auth      *authentication.Auth
	out       output.Outputer
	checkout  *checkout.Checkout
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func NewCheckout(prime primeable) *Checkout {
	return &Checkout{
		prime,
		prime.Auth(),
		prime.Output(),
		checkout.New(git.NewRepo(), prime),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Checkout) Run(params *Params) (rerr error) {
	defer func() { runtime_runbit.RationalizeSolveError(u.prime.Project(), u.auth, &rerr) }()

	logging.Debug("Checkout %v", params.Namespace)

	logging.Debug("Checking out %s to %s", params.Namespace.String(), params.PreferredPath)

	u.out.Notice(locale.Tr("checking_out", params.Namespace.String()))

	var err error
	projectDir, err := u.checkout.Run(params.Namespace, params.Branch, params.RuntimePath, params.PreferredPath, params.NoClone)
	if err != nil {
		return errs.Wrap(err, "Checkout failed")
	}

	proj, err := project.FromPath(projectDir)
	if err != nil {
		return locale.WrapError(err, "err_project_frompath")
	}
	u.prime.SetProject(proj)

	// If an error occurs, remove the created activestate.yaml file and/or directory.
	if !params.Force {
		defer func() {
			if rerr == nil {
				return
			}
			err := os.Remove(proj.Path())
			if err != nil {
				multilog.Error("Failed to remove activestate.yaml after `state checkout` error: %v", err)
				return
			}
			if cwd, err := osutils.Getwd(); err == nil {
				if createdDir := filepath.Dir(proj.Path()); createdDir != cwd {
					err2 := os.RemoveAll(createdDir)
					if err2 != nil {
						multilog.Error("Failed to remove created directory after `state checkout` error: %v", err2)
					}
				}
			}
		}()
	}

	commitID, err := localcommit.Get(proj.Path())
	if err != nil {
		return errs.Wrap(err, "Could not get local commit")
	}

	// Solve runtime
	solveSpinner := output.StartSpinner(u.out, locale.T("progress_solve"), constants.TerminalAnimationInterval)
	bpm := bpModel.NewBuildPlannerModel(u.auth)
	commit, err := bpm.FetchCommit(commitID, proj.Owner(), proj.Name(), nil)
	if err != nil {
		solveSpinner.Stop(locale.T("progress_fail"))
		return errs.Wrap(err, "Failed to fetch build result")
	}
	solveSpinner.Stop(locale.T("progress_success"))

	dependencies.OutputSummary(u.out, commit.BuildPlan().RequestedArtifacts())
	if err := cves.NewCveReport(u.prime).Report(commit.BuildPlan(), nil); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}
	rti, err := runtime_runbit.Update(u.prime, trigger.TriggerCheckout,
		runtime_runbit.WithCommit(commit),
	)
	if err != nil {
		return errs.Wrap(err, "Could not setup runtime")
	}

	var execDir string
	var checkoutStatement string
	if !u.config.GetBool(constants.AsyncRuntimeConfig) {
		execDir = rti.Env(false).ExecutorsPath
		checkoutStatement = locale.Tr("checkout_project_statement", proj.NamespaceString(), proj.Dir(), execDir)
	} else {
		checkoutStatement = locale.Tr("checkout_project_statement_async", proj.NamespaceString(), proj.Dir())
	}

	u.out.Print(output.Prepare(
		checkoutStatement,
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path"`
			Executables string `json:"executables,omitempty"`
		}{
			proj.NamespaceString(),
			proj.Dir(),
			execDir,
		}))

	return nil
}
