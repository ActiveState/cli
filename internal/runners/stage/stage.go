package stage

import (
	"github.com/ActiveState/cli/internal/analytics"
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
}

type Stage struct {
	out       output.Outputer
	proj      *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

func New(p primeable) *Stage {
	return &Stage{
		out:       p.Output(),
		proj:      p.Project(),
		auth:      p.Auth(),
		analytics: p.Analytics(),
		svcModel:  p.SvcModel(),
	}
}

func (s *Stage) Run() error {
	if s.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	changesStaged, err := buildscript.Sync(s.proj, nil, s.out, s.auth)
	if err != nil {
		return locale.WrapError(
			err, "err_stage_sync_buildscript",
			"Could not synchronize the buildscript.",
		)
	}

	trigger := target.TriggerStage
	rti, err := runtime.NewFromProject(s.proj, trigger, s.analytics, s.svcModel, s.out, s.auth)
	if err != nil {
		return locale.WrapInputError(
			err, "err_stage_runtime_new",
			"Could not update runtime for this project.",
		)
	}

	execDir := setup.ExecDir(rti.Target().Dir())

	message := locale.Tl(
		"stage_notice_no_change",
		"No change to the buildscript was found.",
	)
	if changesStaged {
		message = locale.Tl(
			"refresh_project_statement",
			"", s.proj.NamespaceString(), s.proj.Dir(), execDir,
		)
	}
	s.out.Print(output.Prepare(
		message,
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path"`
			Executables string `json:"executables"`
			Staged      bool   `json:"staged"`
		}{
			s.proj.NamespaceString(),
			s.proj.Dir(),
			execDir,
			changesStaged,
		},
	))

	return nil
}