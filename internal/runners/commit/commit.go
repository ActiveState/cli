package commit

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Auther
	primer.Analyticer
	primer.SvcModeler
	primer.Configurer
}

type Commit struct {
	out       output.Outputer
	proj      *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

func New(p primeable) *Commit {
	return &Commit{
		out:       p.Output(),
		proj:      p.Project(),
		auth:      p.Auth(),
		analytics: p.Analytics(),
		svcModel:  p.SvcModel(),
		cfg:       p.Config(),
	}
}

func rationalizeError(err *error) {
	switch {
	case err == nil:
		return
	case errs.Matches(*err, buildscript.ErrBuildscriptNotExist):
		*err = errs.WrapUserFacing(*err, locale.T("err_buildscript_notexist"))
	}
}

func (c *Commit) Run() (rerr error) {
	defer rationalizeError(&rerr)

	if c.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	// Get buildscript.as representation
	script, err := buildscript.ScriptFromProject(c.proj)
	if err != nil {
		return errs.Wrap(err, "Could not get local build script")
	}

	// Get build expression for current state of the project
	localCommitID, err := localcommit.Get(c.proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit ID")
	}
	bp := model.NewBuildPlannerModel(c.auth)
	exprProject, err := bp.GetBuildExpression(localCommitID.String())
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr for provided commit")
	}

	// Check if there is anything to commit
	if script.EqualsBuildExpression(exprProject) {
		c.out.Print(output.Prepare(
			locale.Tl(
				"commit_notice_no_change",
				"No change to the buildscript was found.",
			),
			struct{}{},
		))

		return nil
	}

	exprBuildscript, err := script.BuildExpression()
	if err != nil {
		return errs.Wrap(err, "Unable to get build expression from build script")
	}

	stagedCommitID, err := bp.StageCommit(model.StageCommitParams{
		Owner:        c.proj.Owner(),
		Project:      c.proj.Name(),
		ParentCommit: localCommitID.String(),
		Expression:   exprBuildscript,
	})
	if err != nil {
		return errs.Wrap(err, "Could not update project to reflect build script changes.")
	}

	if err := localcommit.Set(c.proj.Dir(), stagedCommitID.String()); err != nil {
		return errs.Wrap(err, "Could not set local commit ID")
	}

	// Update our local build expression to match the committed one. This allows our API a way to ensure forward compatibility.
	newBuildExpr, err := bp.GetBuildExpression(stagedCommitID.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get the remote build expression")
	}
	if err := buildscript.Update(c.proj, newBuildExpr, c.auth); err != nil {
		return errs.Wrap(err, "Could not update local build script.")
	}

	trigger := target.TriggerCommit
	rti, err := runtime.NewFromProject(c.proj, trigger, c.analytics, c.svcModel, c.out, c.auth, c.cfg)
	if err != nil {
		return locale.WrapInputError(
			err, "err_commit_runtime_new",
			"Could not update runtime for this project.",
		)
	}

	execDir := setup.ExecDir(rti.Target().Dir())

	c.out.Print(output.Prepare(
		locale.Tl(
			"refresh_project_statement",
			"", c.proj.NamespaceString(), c.proj.Dir(), execDir,
		),
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path"`
			Executables string `json:"executables"`
		}{
			c.proj.NamespaceString(),
			c.proj.Dir(),
			execDir,
		},
	))

	return nil
}
