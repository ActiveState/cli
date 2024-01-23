package builds

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
}

type Params struct {
	All bool
}

func NewParams() *Params {
	return &Params{}
}

type Builds struct {
	out     output.Outputer
	project *project.Project
}

type structuredOutput struct {
	Platforms []*structuredPlatform
}

func (o *structuredOutput) MarshalStructured(output.Format) interface{} {
	return o
}

type structuredPlatform struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Builds []*structuredBuild
}

type structuredBuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func New(p primeable) *Builds {
	return &Builds{
		out:     p.Output(),
		project: p.Project(),
	}
}

func rationalizeError(err *error) {
	switch {
	case err == nil:
		return
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())
	}
}

func (b *Builds) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	if b.project == nil {
		return rationalize.ErrNoProject
	}

	runtimeStore := store.New(target.NewProjectTarget(b.project, nil, target.TriggerBuilds).Dir())
	bp, err := runtimeStore.BuildPlan()
	if err != nil {
		return errs.Wrap(err, "Could not get build plan")
	}

	terminalArtfMap, err := buildplan.NewMapFromBuildPlan(bp, false, false, nil)
	if err != nil {
		return errs.Wrap(err, "Could not get build plan map")
	}

	platformMap, err := model.FetchPlatformsMap()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms")
	}

	out := &structuredOutput{}
	for term, artifacts := range terminalArtfMap {
		if !strings.Contains(term, "platform:") {
			continue
		}
		platformUUID := strfmt.UUID(strings.TrimPrefix(term, "platform:"))
		platform, ok := platformMap[platformUUID]
		if !ok {
			return errs.New("Platform does not exist on inventory API: %s", platformUUID)
		}

		p := &structuredPlatform{
			ID:     string(platformUUID),
			Name:   *platform.DisplayName,
			Builds: []*structuredBuild{},
		}
		for _, artifact := range artifacts {
			if artifact.MimeType == bpModel.XActiveStateBuilderMimeType {
				continue
			}
			if !params.All && bpModel.IsStateToolArtifact(artifact.MimeType) {
				continue
			}
			p.Builds = append(p.Builds, &structuredBuild{
				ID:   string(artifact.ArtifactID),
				Name: artifact.Name,
				URL:  artifact.URL,
			})
		}
		out.Platforms = append(out.Platforms, p)
	}

	if b.out.Type().IsStructured() {
		b.out.Print(out)
		return nil
	}

	return b.outputPlain(out)
}

func (b *Builds) outputPlain(out *structuredOutput) error {
	for _, platform := range out.Platforms {
		b.out.Print(fmt.Sprintf("• [NOTICE]%s[/RESET]", platform.Name))
		zeroArtifacts := true
		for _, artifact := range platform.Builds {
			b.out.Print(fmt.Sprintf("  • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, strings.ToUpper(string(artifact.ID)[0:8])))
			zeroArtifacts = false
		}

		if zeroArtifacts {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("no_builds", "No builds")))
		}
	}

	b.out.Print("\nTo download builds run '[ACTIONABLE]state builds dl <ID>[/RESET]'.")
	return nil
}
