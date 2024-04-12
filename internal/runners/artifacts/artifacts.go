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
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
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

	terminalArtfMap, buildComplete, hasFailedArtifacts, err := getTerminalArtifactMap(
		b.project, params.Namespace, params.CommitID, params.Target, b.auth, b.out)
	if err != nil {
		return errs.Wrap(err, "Could not get terminal artifact map")
	}

	platformMap, err := model.FetchPlatformsMap(b.auth)
	if err != nil {
		return errs.Wrap(err, "Could not get platforms")
	}

	out := &StructuredOutput{HasFailedArtifacts: hasFailedArtifacts, BuildComplete: buildComplete}
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
			ID:        string(platformUUID),
			Name:      *platform.DisplayName,
			Artifacts: []*structuredArtifact{},
		}
		for _, artifact := range artifacts {
			if artifact.MimeType == bpResp.XActiveStateBuilderMimeType {
				continue
			}
			if artifact.URL == "" {
				continue
			}
			name := artifact.Name
			if artifact.Version != nil && *artifact.Version != "" {
				name = fmt.Sprintf("%s@%s", name, *artifact.Version)
			}
			build := &structuredArtifact{
				ID:   string(artifact.ArtifactID),
				Name: name,
				URL:  artifact.URL,
			}
			if bpResp.IsStateToolArtifact(artifact.MimeType) {
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

// getTerminalArtifactMap returns a project's terminal artifact map, depending on the given
// arguments. By default, the map for the current project is returned, but a map for a given
// commitID for the current project can be returned, as can the map for a remote project
// (and optional commitID).
func getTerminalArtifactMap(
	pj *project.Project,
	namespace *project.Namespaced,
	commit string,
	target string,
	auth *authentication.Auth,
	out output.Outputer) (_ buildplan.TerminalArtifactMap, completed bool, hasFailedArtifacts bool, rerr error) {
	if pj == nil && !namespace.IsValid() {
		return nil, false, false, rationalize.ErrNoProject
	}

	commitID := strfmt.UUID(commit)
	if commitID != "" && !strfmt.IsUUID(commitID.String()) {
		return nil, false, false, &errInvalidCommitId{errs.New("Invalid commit ID"), commitID.String()}
	}

	namespaceProvided := namespace.IsValid()
	commitIdProvided := commitID != ""

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
	var buildPlan *bpModel.BuildRelay
	switch {
	// Return the artifact map from this runtime.
	case !namespaceProvided && !commitIdProvided:
		localCommitID, err := localcommit.Get(pj.Path())
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Could not get local commit")
		}

		bp := bpModel.NewBuildPlannerModel(auth)
		buildPlan, _, err = bp.FetchBuildResult(localCommitID, pj.Owner(), pj.Name(), targetPtr)
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Failed to fetch build plan")
		}

	// Return artifact map from the given commitID for the current project.
	case !namespaceProvided && commitIdProvided:
		bp := bpModel.NewBuildPlannerModel(auth)
		buildPlan, _, err = bp.FetchBuildResult(commitID, pj.Owner(), pj.Name(), targetPtr)
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Failed to fetch build plan")
		}

	// Return the artifact map for the latest commitID of the given project.
	case namespaceProvided && !commitIdProvided:
		pj, err := model.FetchProjectByName(namespace.Owner, namespace.Project, auth)
		if err != nil {
			return nil, false, false, locale.WrapInputError(err, "err_fetch_project", "", namespace.String())
		}

		branch, err := model.DefaultBranchForProject(pj)
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Could not grab branch for project")
		}

		commitUUID, err := model.BranchCommitID(namespace.Owner, namespace.Project, branch.Label)
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Could not get commit ID for project")
		}
		commitID = *commitUUID

		bp := bpModel.NewBuildPlannerModel(auth)
		buildPlan, _, err = bp.FetchBuildResult(commitID, namespace.Owner, namespace.Project, targetPtr)
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Failed to fetch build plan")
		}

	// Return the artifact map for the given commitID of the given project.
	case namespaceProvided && commitIdProvided:
		bp := bpModel.NewBuildPlannerModel(auth)
		buildPlan, _, err = bp.FetchBuildResult(commitID, namespace.Owner, namespace.Project, targetPtr)
		if err != nil {
			return nil, false, false, errs.Wrap(err, "Failed to fetch build plan")
		}

	default:
		return nil, false, false, errs.New("Unhandled case")
	}

	bpm, err := buildplan.NewMapFromBuildPlan(buildPlan.Build, false, false, nil, true)
	if err != nil {
		return nil, buildPlan.BuildReady, false, errs.Wrap(err, "Could not get buildplan")
	}

	// Communicate whether there were failed artifacts
	for _, artifact := range buildPlan.Build.Artifacts {
		if !bpResp.IsSuccessArtifactStatus(artifact.Status) {
			return bpm, buildPlan.BuildReady, true, nil
		}
	}

	return bpm, buildPlan.BuildReady, false, nil
}
