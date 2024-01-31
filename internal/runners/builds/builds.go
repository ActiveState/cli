package builds

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
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
	primer.SvcModeler
	primer.Configurer
	primer.Analyticer
}

type Params struct {
	All bool
}

type Builds struct {
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *auth.Auth
	config    *config.Instance
}

type StructuredOutput struct {
	Platforms []*structuredPlatform `json:"platforms"`
}

func (o *StructuredOutput) MarshalStructured(output.Format) interface{} {
	return o
}

type structuredPlatform struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Builds   []*structuredBuild `json:"builds"`
	Packages []*structuredBuild `json:"packages"`
}

type structuredBuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func New(p primeable) *Builds {
	return &Builds{
		out:       p.Output(),
		project:   p.Project(),
		auth:      p.Auth(),
		svcModel:  p.SvcModel(),
		config:    p.Config(),
		analytics: p.Analytics(),
	}
}

func (b *Builds) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	if b.project == nil {
		return rationalize.ErrNoProject
	}

	// We don't use the runtime returned here, because builds needs more advanced runtime info, but we still want to call
	// it in case our runtime isn't up-to date.
	_, err := runtime.NewFromProject(b.project, target.TriggerBuilds, b.analytics, b.svcModel, b.out, b.auth, b.config)
	if err != nil {
		return locale.WrapInputError(err, "err_refresh_runtime_new", "Could not update runtime for this project.")
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

	out := &StructuredOutput{}
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
			name := artifact.Name
			if artifact.Version != nil && *artifact.Version != "" {
				name = fmt.Sprintf("%s@%s", name, *artifact.Version)
			}
			build := &structuredBuild{
				ID:   string(artifact.ArtifactID),
				Name: name,
				URL:  artifact.URL,
			}
			if bpModel.IsStateToolArtifact(artifact.MimeType) {
				if !params.All {
					continue
				}
				p.Packages = append(p.Packages, build)
			} else {
				p.Builds = append(p.Builds, build)
			}
		}
		sort.Slice(p.Builds, func(i, j int) bool {
			return strings.ToLower(p.Builds[i].Name) < strings.ToLower(p.Builds[j].Name)
		})
		sort.Slice(p.Packages, func(i, j int) bool {
			return strings.ToLower(p.Packages[i].Name) < strings.ToLower(p.Packages[j].Name)
		})
		out.Platforms = append(out.Platforms, p)
	}

	sort.Slice(out.Platforms, func(i, j int) bool {
		return strings.ToLower(out.Platforms[i].Name) < strings.ToLower(out.Platforms[j].Name)
	})

	if b.out.Type().IsStructured() {
		b.out.Print(out)
		return nil
	}

	return b.outputPlain(out)
}

func (b *Builds) outputPlain(out *StructuredOutput) error {
	for _, platform := range out.Platforms {
		b.out.Print(fmt.Sprintf("• [NOTICE]%s[/RESET]", platform.Name))
		for _, artifact := range platform.Builds {
			b.out.Print(fmt.Sprintf("  • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, strings.ToUpper(string(artifact.ID)[0:8])))
		}

		if len(platform.Packages) > 0 {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("builds_packages", "[NOTICE]Packages[/RESET]")))
		}
		for _, artifact := range platform.Packages {
			b.out.Print(fmt.Sprintf("    • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, strings.ToUpper(string(artifact.ID)[0:8])))
		}

		if len(platform.Builds) == 0 && len(platform.Packages) == 0 {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("no_builds", "No builds")))
		}
	}

	b.out.Print("\nTo download builds run '[ACTIONABLE]state builds dl <ID>[/RESET]'.")
	return nil
}
