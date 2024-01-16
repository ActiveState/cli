package commit

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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

func (c *Commit) Run() error {
	if c.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	changesCommitted, err := buildscript.Sync(c.proj, nil, c.out, c.auth)
	if err != nil {
		return locale.WrapError(
			err, "err_commit_sync_buildscript",
			"Could not synchronize the buildscript.",
		)
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

	if !changesCommitted {
		c.out.Print(output.Prepare(
			locale.Tl(
				"commit_notice_no_change",
				"No change to the buildscript was found.",
			),
			struct{}{},
		))

		return nil
	}

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
