package pull

import (
	"os"
	"path"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/hail"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Pull struct {
	project *project.Project
	out     output.Outputer
}

type primeable interface {
	primer.Projecter
	primer.Outputer
}

func New(prime primeable) *Pull {
	return &Pull{
		prime.Project(),
		prime.Output(),
	}
}

type outputFormat struct {
	Message string `locale:"message,Message"`
	Success bool   `locale:"success,Success"`
}

func (f *outputFormat) MarshalOutput(format output.Format) interface{} {
	switch format {
	case output.EditorV0FormatName:
		return f.editorV0Format()
	case output.PlainFormatName:
		return f.Message
	}

	return f
}

func (p *Pull) Run() error {
	if p.project == nil {
		return locale.NewInputError("err_no_project")
	}

	if p.project.IsHeadless() {
		return locale.NewInputError("err_pull_headless", "You must first create a project. Please visit {{.V0}} to create your project.", p.project.URL())
	}

	// Retrieve latest commit ID on platform
	latestID, err := model.LatestCommitID(p.project.Owner(), p.project.Name())
	if err != nil {
		return locale.WrapInputError(err, "err_pull_commit", "Could not retrieve the latest commit for your project.")
	}

	// Update the commit ID in the activestate.yaml
	if p.project.CommitID() != latestID.String() {
		err := p.project.Source().SetCommit(latestID.String(), false)
		if err != nil {
			return locale.WrapError(err, "err_pull_update", "Cannot update the commit in your project file.")
		}

		p.out.Print(&outputFormat{
			locale.T("pull_updated"),
			true,
		})
	} else {
		p.out.Print(&outputFormat{
			locale.Tl("pull_not_updated", "Your activestate.yaml is already up to date."),
			false,
		})
	}

	actID := os.Getenv(constants.ActivatedStateIDEnvVarName)
	if actID == "" {
		logging.Debug("Not in an activated environment, so no need to reactivate")
		return nil
	}

	fname := path.Join(config.Get().ConfigPath(), constants.UpdateHailFileName)
	// must happen last in this function scope (defer if needed)
	if err := hail.Send(fname, []byte(actID)); err != nil {
		logging.Error("failed to send hail via %q: %s", fname, err)
		return locale.WrapError(err, "err_pull_hail", "Could not re-activate your project, please exit and re-activate manually by running 'state activate' again.")
	}

	return nil
}
