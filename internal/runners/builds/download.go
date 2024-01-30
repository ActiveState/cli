package builds

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type DownloadParams struct {
	BuildID   string
	OutputDir string
}

type Download struct {
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *auth.Auth
	config    *config.Instance
}

func NewDownload(prime primeable) *Download {
	return &Download{
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		auth:      prime.Auth(),
		config:    prime.Config(),
	}
}

func (d *Download) Run(params *DownloadParams) (rerr error) {
	defer rationalizeError(&rerr)

	if d.project == nil {
		return rationalize.ErrNoProject
	}
	// Source the build plan
	// Find the given node ID in the artifact list
	// Use the artifact URL to download the artifact
	pg := runbits.NewRuntimeProgressIndicator(d.out)
	defer rtutils.Closer(pg.Close, &rerr)
	return nil
}
