package artifacts

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/output/renderers"
	"github.com/ActiveState/cli/internal/primer"
	buildplanner_runbit "github.com/ActiveState/cli/internal/runbits/buildplanner"
	"github.com/ActiveState/cli/pkg/buildplan"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/google/uuid"
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
	All       bool
	Namespace *project.Namespaced
	CommitID  string
	Target    string
	Full      bool
}

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

type Artifacts struct {
	prime     primeable
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
	config    *config.Instance
}

type StructuredOutput struct {
	BuildComplete      bool                  `json:"build_completed"`
	HasFailedArtifacts bool                  `json:"has_failed_artifacts"`
	Platforms          []*structuredPlatform `json:"platforms"`
}

func (o *StructuredOutput) MarshalStructured(output.Format) interface{} {
	return o
}

type structuredPlatform struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Artifacts []*structuredArtifact `json:"artifacts"`
	Packages  []*structuredArtifact `json:"packages"`
}

type structuredArtifact struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	status string
	Errors []string `json:"errors,omitempty"`
	LogURL string   `json:"logUrl,omitempty"`
}

func New(p primeable) *Artifacts {
	return &Artifacts{
		prime:     p,
		out:       p.Output(),
		project:   p.Project(),
		auth:      p.Auth(),
		svcModel:  p.SvcModel(),
		config:    p.Config(),
		analytics: p.Analytics(),
	}
}

func rationalizeArtifactsError(proj *project.Project, auth *authentication.Auth, rerr *error) {
	if rerr == nil {
		return
	}

	var planningError *bpResp.BuildPlannerError
	switch {
	case errors.As(*rerr, &planningError):
		// Forward API error to user.
		*rerr = errs.WrapUserFacing(*rerr, planningError.Error())

	default:
		rationalizeCommonError(proj, auth, rerr)
	}
}

func (b *Artifacts) Run(params *Params) (rerr error) {
	defer rationalizeArtifactsError(b.project, b.auth, &rerr)

	if b.project != nil && !params.Namespace.IsValid() {
		b.out.Notice(locale.Tr("operating_message", b.project.NamespaceString(), b.project.Dir()))
	}

	bp, err := buildplanner_runbit.GetBuildPlan(
		params.Namespace, params.CommitID, params.Target, b.prime)
	if err != nil {
		return errs.Wrap(err, "Could not get buildplan")
	}

	platformMap, err := model.FetchPlatformsMap()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms")
	}

	hasFailedArtifacts := len(bp.Artifacts(buildplan.FilterFailedArtifacts())) > 0

	out := &StructuredOutput{HasFailedArtifacts: hasFailedArtifacts, BuildComplete: bp.IsBuildReady()}
	for _, platformUUID := range bp.Platforms() {
		platform, ok := platformMap[platformUUID]
		if !ok {
			return errs.New("Platform does not exist on inventory API: %s", platformUUID)
		}
		p := &structuredPlatform{
			ID:        string(platformUUID),
			Name:      *platform.DisplayName,
			Artifacts: []*structuredArtifact{},
		}
		for _, artifact := range bp.Artifacts(buildplan.FilterPlatformArtifacts(platformUUID)) {
			if artifact.MimeType == types.XActiveStateBuilderMimeType {
				continue
			}
			name := artifact.Name()

			// Detect and drop artifact names which start with a uuid, as this isn't user friendly
			nameBits := strings.Split(name, " ")
			if len(nameBits) > 1 {
				if _, err := uuid.Parse(nameBits[0]); err == nil {
					name = fmt.Sprintf("%s (%s)", strings.Join(nameBits[1:], " "), filepath.Base(artifact.URL))
				}
			}

			version := artifact.Version()
			if version != "" {
				name = fmt.Sprintf("%s@%s", name, version)
			}

			build := &structuredArtifact{
				ID:     string(artifact.ArtifactID),
				Name:   name,
				URL:    artifact.URL,
				status: artifact.Status,
				Errors: artifact.Errors,
				LogURL: artifact.LogURL,
			}
			if bpModel.IsStateToolArtifact(artifact.MimeType) {
				if !params.All {
					continue
				}
				p.Packages = append(p.Packages, build)
			} else {
				p.Artifacts = append(p.Artifacts, build)
			}
		}
		sort.Slice(p.Artifacts, func(i, j int) bool {
			return strings.ToLower(p.Artifacts[i].Name) < strings.ToLower(p.Artifacts[j].Name)
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

	return b.outputPlain(out, params.Full)
}

func (b *Artifacts) outputPlain(out *StructuredOutput, fullID bool) error {
	for _, platform := range out.Platforms {
		b.out.Print(fmt.Sprintf("• [NOTICE]%s[/RESET]", platform.Name))
		for _, artifact := range platform.Artifacts {
			switch {
			case len(artifact.Errors) > 0:
				b.out.Print(renderers.NewBulletList("  • ",
					renderers.HeadedBulletTree,
					[]string{
						fmt.Sprintf("%s ([ERROR]%s[/RESET])", artifact.Name, locale.T("artifact_status_failed")),
						fmt.Sprintf("%s: [ERROR]%s[/RESET]", locale.T("artifact_status_failed_message"), strings.Join(artifact.Errors, ": ")),
						fmt.Sprintf("%s: [ACTIONABLE]%s[/RESET]", locale.T("artifact_status_failed_log"), artifact.LogURL),
					}))
				continue
			case artifact.status == types.ArtifactSkipped:
				b.out.Print(fmt.Sprintf("  • %s ([NOTICE]%s[/RESET])", artifact.Name, locale.T("artifact_status_skipped")))
				continue
			case artifact.URL == "":
				b.out.Print(fmt.Sprintf("  • %s ([WARNING]%s ...[/RESET])", artifact.Name, locale.T("artifact_status_building")))
				continue
			}
			id := strings.ToUpper(artifact.ID)
			if !fullID {
				id = id[0:8]
			}
			b.out.Print(fmt.Sprintf("  • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, id))
		}

		if len(platform.Packages) > 0 {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("artifacts_packages", "[NOTICE]Packages[/RESET]")))
		}
		for _, artifact := range platform.Packages {
			switch {
			case len(artifact.Errors) > 0:
				b.out.Print(renderers.NewBulletList("    • ",
					renderers.HeadedBulletTree,
					[]string{
						fmt.Sprintf("%s ([ERROR]%s[/RESET])", artifact.Name, locale.T("artifact_status_failed")),
						fmt.Sprintf("%s: [ERROR]%s[/RESET]", locale.T("artifact_status_failed_message"), strings.Join(artifact.Errors, ": ")),
						fmt.Sprintf("%s: [ACTIONABLE]%s[/RESET]", locale.T("artifact_status_failed_log"), artifact.LogURL),
					}))
				continue
			case artifact.status == types.ArtifactSkipped:
				b.out.Print(fmt.Sprintf("    • %s ([NOTICE]%s[/RESET])", artifact.Name, locale.T("artifact_status_skipped")))
				continue
			case artifact.URL == "":
				b.out.Print(fmt.Sprintf("    • %s ([WARNING]%s ...[/RESET])", artifact.Name, locale.T("artifact_status_building")))
				continue
			}
			id := strings.ToUpper(artifact.ID)
			if !fullID {
				id = id[0:8]
			}
			b.out.Print(fmt.Sprintf("    • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, id))
		}

		if len(platform.Artifacts) == 0 && len(platform.Packages) == 0 {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("no_artifacts", "No artifacts")))
		}
	}

	if out.HasFailedArtifacts {
		b.out.Notice("") // blank line
		b.out.Error(locale.T("warn_has_failed_artifacts"))
	}

	if !out.BuildComplete {
		b.out.Notice("") // blank line
		b.out.Notice(locale.T("warn_build_not_complete"))
	}

	b.out.Print("\nTo download artifacts run '[ACTIONABLE]state artifacts dl <ID>[/RESET]'.")
	return nil
}
