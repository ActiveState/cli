package builds

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
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
	All       bool
	Namespace *project.Namespaced
	CommitID  string
	Full      bool
}

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

type Builds struct {
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	auth      *authentication.Auth
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

type errInvalidCommitId struct {
	error
	id string
}

func rationalizeBuildsError(err *error, auth *authentication.Auth) {
	switch {
	case err == nil:
		return
	default:
		rationalizeCommonError(err, auth)
	}
}

func (b *Builds) Run(params *Params) (rerr error) {
	defer rationalizeBuildsError(&rerr, b.auth)

	if b.project != nil && !params.Namespace.IsValid() {
		b.out.Notice(locale.Tr("operating_message", b.project.NamespaceString(), b.project.Dir()))
	}

	terminalArtfMap, err := getTerminalArtifactMap(
		b.project, params.Namespace, params.CommitID, b.auth, b.analytics, b.svcModel, b.out, b.config)
	if err != nil {
		return errs.Wrap(err, "Could not get terminal artifact map")
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

	return b.outputPlain(out, params.Full)
}

func (b *Builds) outputPlain(out *StructuredOutput, fullID bool) error {
	for _, platform := range out.Platforms {
		b.out.Print(fmt.Sprintf("• [NOTICE]%s[/RESET]", platform.Name))
		for _, artifact := range platform.Builds {
			id := strings.ToUpper(artifact.ID)
			if !fullID {
				id = id[0:8]
			}
			b.out.Print(fmt.Sprintf("  • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, id))
		}

		if len(platform.Packages) > 0 {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("builds_packages", "[NOTICE]Packages[/RESET]")))
		}
		for _, artifact := range platform.Packages {
			id := strings.ToUpper(artifact.ID)
			if !fullID {
				id = id[0:8]
			}
			b.out.Print(fmt.Sprintf("    • %s (ID: [ACTIONABLE]%s[/RESET])", artifact.Name, id))
		}

		if len(platform.Builds) == 0 && len(platform.Packages) == 0 {
			b.out.Print(fmt.Sprintf("  • %s", locale.Tl("no_builds", "No builds")))
		}
	}

	b.out.Print("\nTo download builds run '[ACTIONABLE]state builds dl <ID>[/RESET]'.")
	return nil
}

// getTerminalArtifactMap returns a project's terminal artifact map, depending on the given
// arguments. By default, the map for the current project is returned, but a map for a given
// commitID for the current project can be returned, as can the map for a remote project
// (and optional commitID).
func getTerminalArtifactMap(
	pj *project.Project,
	namespace *project.Namespaced,
	commit string,
	auth *authentication.Auth,
	an analytics.Dispatcher,
	svcModel *model.SvcModel,
	out output.Outputer,
	cfg Configurable) (_ buildplan.TerminalArtifactMap, rerr error) {
	if pj == nil && !namespace.IsValid() {
		return nil, rationalize.ErrNoProject
	}

	commitID := strfmt.UUID(commit)
	if commitID != "" && !strfmt.IsUUID(commitID.String()) {
		return nil, &errInvalidCommitId{errs.New("Invalid commit ID"), commitID.String()}
	}

	namespaceProvided := namespace.IsValid()
	commitIdProvided := commitID != ""

	if namespaceProvided || commitIdProvided {
		// Show a spinner when fetching a terminal artifact map.
		// Sourcing the local runtime for an artifact map has its own spinner.
		pb := output.StartSpinner(out, locale.T("progress_solve"), constants.TerminalAnimationInterval)
		defer func() {
			message := locale.T("progress_success")
			if rerr != nil {
				message = locale.T("progress_fail")
			}
			pb.Stop(message + "\n") // extra empty line
		}()
	}

	switch {
	// Return the artifact map from this runtime.
	case !namespaceProvided && !commitIdProvided:
		rt, err := runtime.NewFromProject(pj, nil, target.TriggerBuilds, an, svcModel, out, auth, cfg)
		if err != nil {
			return nil, locale.WrapInputError(err, "err_refresh_runtime_new", "Could not update runtime for this project.")
		}
		return rt.TerminalArtifactMap(false)

	// Return artifact map from the given commitID for the current project.
	case !namespaceProvided && commitIdProvided:
		bp := model.NewBuildPlannerModel(auth)
		buildPlan, err := bp.FetchBuildResult(commitID, pj.Owner(), pj.Name())
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch build plan")
		}
		return buildplan.NewMapFromBuildPlan(buildPlan.Build, false, false, nil)

	// Return the artifact map for the latest commitID of the given project.
	case namespaceProvided && !commitIdProvided:
		pj, err := model.FetchProjectByName(namespace.Owner, namespace.Project)
		if err != nil {
			return nil, locale.WrapInputError(err, "err_fetch_project", "", namespace.String())
		}

		branch, err := model.DefaultBranchForProject(pj)
		if err != nil {
			return nil, errs.Wrap(err, "Could not grab branch for project")
		}

		commitUUID, err := model.BranchCommitID(namespace.Owner, namespace.Project, branch.Label)
		if err != nil {
			return nil, errs.Wrap(err, "Could not get commit ID for project")
		}
		commitID = *commitUUID

		bp := model.NewBuildPlannerModel(auth)
		buildPlan, err := bp.FetchBuildResult(commitID, namespace.Owner, namespace.Project)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch build plan")
		}
		return buildplan.NewMapFromBuildPlan(buildPlan.Build, false, false, nil)

	// Return the artifact map for the given commitID of the given project.
	case namespaceProvided && commitIdProvided:
		bp := model.NewBuildPlannerModel(auth)
		buildPlan, err := bp.FetchBuildResult(commitID, namespace.Owner, namespace.Project)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch build plan")
		}
		return buildplan.NewMapFromBuildPlan(buildPlan.Build, false, false, nil)

	default:
		return nil, errs.New("Unhandled case")
	}
}
