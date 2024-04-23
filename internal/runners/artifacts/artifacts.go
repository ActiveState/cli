package artifacts

import (
	"errors"
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
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
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
	Target    string
	Full      bool
}

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

type Artifacts struct {
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
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func New(p primeable) *Artifacts {
	return &Artifacts{
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

func rationalizeArtifactsError(rerr *error, auth *authentication.Auth) {
	if rerr == nil {
		return
	}

	var planningError *bpResp.BuildPlannerError
	switch {
	case errors.As(*rerr, &planningError):
		// Forward API error to user.
		*rerr = errs.WrapUserFacing(*rerr, planningError.Error())

	default:
		rationalizeCommonError(rerr, auth)
	}
}

func (b *Artifacts) Run(params *Params) (rerr error) {
	defer rationalizeArtifactsError(&rerr, b.auth)

	if b.project != nil && !params.Namespace.IsValid() {
		b.out.Notice(locale.Tr("operating_message", b.project.NamespaceString(), b.project.Dir()))
	}

	bp, err := getBuildPlan(
		b.project, params.Namespace, params.CommitID, params.Target, b.auth, b.out)
	if err != nil {
		return errs.Wrap(err, "Could not get terminal artifact map")
	}

	platformMap, err := model.FetchPlatformsMap()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms")
	}

	hasFailedArtifacts := len(bp.Artifacts()) != len(bp.Artifacts(buildplan.FilterSuccessfulArtifacts()))

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
			if artifact.URL == "" {
				continue
			}
			name := artifact.Name()
			version := artifact.Version()
			if version != "" {
				name = fmt.Sprintf("%s@%s", name, version)
			}
			build := &structuredArtifact{
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
	if !out.BuildComplete {
		b.out.Error(locale.T("warn_build_not_complete"))
	}
	if out.HasFailedArtifacts {
		b.out.Error(locale.T("warn_has_failed_artifacts"))
	}

	for _, platform := range out.Platforms {
		b.out.Print(fmt.Sprintf("• [NOTICE]%s[/RESET]", platform.Name))
		for _, artifact := range platform.Artifacts {
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

	b.out.Print("\nTo download artifacts run '[ACTIONABLE]state artifacts dl <ID>[/RESET]'.")
	return nil
}

// getBuildPlan returns a project's terminal artifact map, depending on the given
// arguments. By default, the map for the current project is returned, but a map for a given
// commitID for the current project can be returned, as can the map for a remote project
// (and optional commitID).
func getBuildPlan(
	pj *project.Project,
	namespace *project.Namespaced,
	commitID string,
	target string,
	auth *authentication.Auth,
	out output.Outputer) (bp *buildplan.BuildPlan, rerr error) {
	if pj == nil && !namespace.IsValid() {
		return nil, rationalize.ErrNoProject
	}

	commitUUID := strfmt.UUID(commitID)
	if commitUUID != "" && !strfmt.IsUUID(commitUUID.String()) {
		return nil, &errInvalidCommitId{errs.New("Invalid commit ID"), commitUUID.String()}
	}

	namespaceProvided := namespace.IsValid()
	commitIdProvided := commitUUID != ""

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

	targetPtr := ptr.To(request.TargetAll)
	if target != "" {
		targetPtr = &target
	}

	var err error
	var commit *bpModel.Commit
	switch {
	// Return the artifact map from this runtime.
	case !namespaceProvided && !commitIdProvided:
		localCommitID, err := localcommit.Get(pj.Path())
		if err != nil {
			return nil, errs.Wrap(err, "Could not get local commit")
		}

		bp := bpModel.NewBuildPlannerModel(auth)
		commit, err = bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	// Return artifact map from the given commitID for the current project.
	case !namespaceProvided && commitIdProvided:
		bp := bpModel.NewBuildPlannerModel(auth)
		commit, err = bp.FetchCommit(commitUUID, pj.Owner(), pj.Name(), targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	// Return the artifact map for the latest commitID of the given project.
	case namespaceProvided && !commitIdProvided:
		pj, err := model.FetchProjectByName(namespace.Owner, namespace.Project, auth)
		if err != nil {
			return nil, locale.WrapInputError(err, "err_fetch_project", "", namespace.String())
		}

		branch, err := model.DefaultBranchForProject(pj)
		if err != nil {
			return nil, errs.Wrap(err, "Could not grab branch for project")
		}

		branchCommitUUID, err := model.BranchCommitID(namespace.Owner, namespace.Project, branch.Label)
		if err != nil {
			return nil, errs.Wrap(err, "Could not get commit ID for project")
		}
		commitUUID = *branchCommitUUID

		bp := bpModel.NewBuildPlannerModel(auth)
		commit, err = bp.FetchCommit(commitUUID, namespace.Owner, namespace.Project, targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	// Return the artifact map for the given commitID of the given project.
	case namespaceProvided && commitIdProvided:
		bp := bpModel.NewBuildPlannerModel(auth)
		commit, err = bp.FetchCommit(commitUUID, namespace.Owner, namespace.Project, targetPtr)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to fetch commit")
		}

	default:
		return nil, errs.New("Unhandled case")
	}

	return commit.BuildPlan(), nil
}
