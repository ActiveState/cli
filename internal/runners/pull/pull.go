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
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Pull struct {
	project *project.Project
	out     output.Outputer
}

func New(pj *project.Project, out output.Outputer) *Pull {
	return &Pull{
		pj,
		out,
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
		return locale.NewInputError("err_pull_noproject", "You have to be inside a project folder to be able to pull in updates. Project folders contain an activestate.yaml.")
	}

	// Retrieve latest commit ID on platform
	latestID, fail := model.LatestCommitID(p.project.Owner(), p.project.Name())
	if fail != nil {
		return locale.WrapInputError(fail, "err_pull_commit", "Could not retrieve the latest commit for your project.")
	}

	// Update the commit ID in the activestate.yaml
	if p.project.CommitID() != latestID.String() {
		fail := p.project.Source().SetCommit(latestID.String())
		if fail != nil {
			return locale.WrapError(fail, "err_pull_update", "Cannot update the commit in your project file.")
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

	fname := path.Join(config.ConfigPath(), constants.UpdateHailFileName)
	// must happen last in this function scope (defer if needed)
	if fail := hail.Send(fname, []byte(actID)); fail != nil {
		logging.Error("failed to send hail via %q: %s", fname, fail)
		return locale.WrapError(fail, "err_pull_hail", "Could not re-activate your project, please exit and re-activate manually by running 'state activate' again.")
	}

	return nil
}
