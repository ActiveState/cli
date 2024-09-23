package upgrade

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/commits_runbit"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/table"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/go-openapi/strfmt"
)

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
	primer.Prompter
}

type Params struct {
	Timestamp captain.TimeValue
	Expand    bool
}

func NewParams() *Params {
	return &Params{}
}

type Upgrade struct {
	prime primeable
}

func New(p primeable) *Upgrade {
	return &Upgrade{
		prime: p,
	}
}

var ErrNoChanges = errors.New("no changes")
var ErrAbort = errors.New("aborted")

func rationalizeError(err *error) {
	switch {
	case err == nil:
		return
	case errors.Is(*err, ErrNoChanges):
		*err = errs.WrapUserFacing(*err, locale.T("upgrade_no_changes"), errs.SetInput())
	case errors.Is(*err, ErrAbort):
		*err = errs.WrapUserFacing(*err, locale.T("upgrade_aborted"), errs.SetInput())
	}
}

func (u *Upgrade) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	// Validate project
	proj := u.prime.Project()
	if proj == nil {
		return rationalize.ErrNoProject
	}
	if proj.IsHeadless() {
		return rationalize.ErrHeadless
	}

	out := u.prime.Output()
	out.Notice(locale.Tr("operating_message", proj.NamespaceString(), proj.Dir()))

	// Collect buildplans for before/after upgrade
	pg := output.StartSpinner(out, locale.T("upgrade_solving"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	// Collect "before" buildplan
	localCommitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Failed to get local commit")
	}

	bpm := bpModel.NewBuildPlannerModel(u.prime.Auth())
	localCommit, err := bpm.FetchCommit(localCommitID, proj.Owner(), proj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result")
	}

	// Collect "after" buildplan
	bumpedBS, err := localCommit.BuildScript().Clone()
	if err != nil {
		return errs.Wrap(err, "Failed to clone build script")
	}
	ts, err := commits_runbit.ExpandTime(&params.Timestamp, u.prime.Auth())
	if err != nil {
		return errs.Wrap(err, "Failed to fetch latest timestamp")
	}
	bumpedBS.SetAtTime(ts)

	// Since our platform is commit based we need to create a commit for the "after" buildplan, even though we may not
	// end up using it it the user doesn't confirm the upgrade.
	bumpedCommit, err := bpm.StageCommit(bpModel.StageCommitParams{
		Owner:        proj.Owner(),
		Project:      proj.Name(),
		ParentCommit: localCommitID.String(),
		Script:       bumpedBS,
	})
	if err != nil {
		// The buildplanner itself can assert that there are no new changes, in which case we don't want to handle
		// this as an error
		var commitErr *response.CommitError
		if errors.As(err, &commitErr) {
			if commitErr.Type == types.NoChangeSinceLastCommitErrorType {
				pg.Stop(locale.T("progress_success"))
				pg = nil
				return ErrNoChanges
			}
		}
		return errs.Wrap(err, "Failed to stage bumped commit")
	}
	bumpedBP := bumpedCommit.BuildPlan()

	// All done collecting buildplans
	pg.Stop(locale.T("progress_success"))
	pg = nil

	changeset := bumpedBP.DiffArtifacts(localCommit.BuildPlan(), false)
	if len(changeset.Filter(buildplan.ArtifactUpdated)) == 0 {
		// In most cases we would've already reached this error due to the commit failing. But it is possible for
		// the commit to be created (ie. there were changes), but without those changes being relevant to any artifacts
		// that we care about.
		return ErrNoChanges
	}

	changes := u.calculateChanges(changeset, bumpedCommit)
	if out.Type().IsStructured() {
		out.Print(output.Structured(changes))
	} else {
		if err := u.renderUserFacing(changes, params.Expand); err != nil {
			return errs.Wrap(err, "Failed to render user facing upgrade")
		}
	}

	if err := localcommit.Set(u.prime.Project().Dir(), bumpedCommit.CommitID.String()); err != nil {
		return errs.Wrap(err, "Failed to set local commit")
	}

	out.Notice(locale.Tr("upgrade_success"))

	return nil
}

type structuredChange struct {
	Type           string             `json:"type"`
	Name           string             `json:"name"`
	Namespace      string             `json:"namespace,omitempty"`
	OldVersion     string             `json:"old_version,omitempty"`
	NewVersion     string             `json:"new_version"`
	OldRevision    int                `json:"old_revision"`
	NewRevision    int                `json:"new_revision"`
	OldLicenses    []string           `json:"old_licenses,omitempty"`
	NewLicenses    []string           `json:"new_licenses"`
	TransitiveDeps []structuredChange `json:"transitive_dependencies,omitempty"`
}

func (u *Upgrade) calculateChanges(changedArtifacts buildplan.ArtifactChangeset, bumpedCommit *bpModel.Commit) []structuredChange {
	requested := bumpedCommit.BuildPlan().RequestedArtifacts().ToIDMap()

	relevantChanges := changedArtifacts.Filter(buildplan.ArtifactUpdated)
	relevantRequestedArtifacts := buildplan.Artifacts{}

	// Calculate relevant artifacts ahead of time, as we'll need them to calculate transitive dependencies
	// (we want to avoid recursing into the same artifact multiple times)
	for _, change := range relevantChanges {
		if _, ok := requested[change.Artifact.ArtifactID]; !ok {
			continue
		}
		relevantRequestedArtifacts = append(relevantRequestedArtifacts, change.Artifact)
	}

	changes := []structuredChange{}
	for _, artifactUpdate := range changedArtifacts.Filter(buildplan.ArtifactUpdated) {
		if _, ok := requested[artifactUpdate.Artifact.ArtifactID]; !ok {
			continue
		}

		change := structuredChange{
			Type:        artifactUpdate.ChangeType.String(),
			Name:        artifactUpdate.Artifact.Name(),
			OldVersion:  artifactUpdate.Old.Version(),
			NewVersion:  artifactUpdate.Artifact.Version(),
			OldRevision: artifactUpdate.Old.Revision(),
			NewRevision: artifactUpdate.Artifact.Revision(),
			OldLicenses: artifactUpdate.Old.Licenses(),
			NewLicenses: artifactUpdate.Artifact.Licenses(),
		}

		if len(artifactUpdate.Artifact.Ingredients) == 1 {
			change.Namespace = artifactUpdate.Artifact.Ingredients[0].Namespace
		}

		changedDeps := calculateChangedDeps(artifactUpdate.Artifact, relevantRequestedArtifacts, changedArtifacts)
		if len(changedDeps) > 0 {
			change.TransitiveDeps = make([]structuredChange, len(changedDeps))
			for n, changedDep := range changedDeps {
				change.TransitiveDeps[n] = structuredChange{
					Type:        changedDep.ChangeType.String(),
					Name:        changedDep.Artifact.Name(),
					NewVersion:  changedDep.Artifact.Version(),
					NewRevision: changedDep.Artifact.Revision(),
					NewLicenses: changedDep.Artifact.Licenses(),
				}
				if changedDep.Old != nil {
					change.TransitiveDeps[n].OldVersion = changedDep.Old.Version()
					change.TransitiveDeps[n].OldRevision = changedDep.Old.Revision()
					change.TransitiveDeps[n].OldLicenses = changedDep.Old.Licenses()
				}
			}
		}

		changes = append(changes, change)
	}

	return changes
}

func (u *Upgrade) renderUserFacing(changes []structuredChange, expand bool) error {
	out := u.prime.Output()

	out.Notice("") // Empty line

	tbl := table.New(locale.Ts("name", "version", "license"))
	tbl.HideDash = true
	for _, change := range changes {
		tbl.AddRow([]string{
			change.Name,
			renderVersionChange(change),
			renderLicenseChange(change),
		})

		needsDepRow := len(change.TransitiveDeps) > 0
		needsNamespaceRow := strings.HasPrefix(change.Namespace, model.NamespaceOrg.Prefix())

		if needsNamespaceRow {
			treeSymbol := output.TreeEnd
			if needsDepRow {
				treeSymbol = output.TreeMid
			}
			tbl.AddRow([]string{locale.Tr("namespace_row", treeSymbol, change.Namespace)})
		}

		if needsDepRow {
			if expand {
				for n, changedDep := range change.TransitiveDeps {
					treeSymbol := output.TreeEnd
					if n != len(change.TransitiveDeps)-1 {
						treeSymbol = output.TreeMid
					}
					tbl.AddRow([]string{locale.Tr("dependency_detail_row", treeSymbol, changedDep.Name, renderVersionChange(changedDep))})
				}
			} else {
				tbl.AddRow([]string{locale.Tr("dependency_row", output.TreeEnd, strconv.Itoa(len(change.TransitiveDeps)))})
			}
		}
	}

	out.Print(tbl.Render())

	out.Notice(" ") // Empty line (prompts use Notice)
	confirm, err := u.prime.Prompt().Confirm("", locale.Tr("upgrade_confirm"), ptr.To(true))
	if err != nil {
		return errs.Wrap(err, "confirmation failed")
	}
	if !confirm {
		return ErrAbort
	}

	return nil
}

func calculateChangedDeps(artifact *buildplan.Artifact, dontCount buildplan.Artifacts, changeset buildplan.ArtifactChangeset) buildplan.ArtifactChangeset {
	ignore := map[strfmt.UUID]struct{}{}
	for _, skip := range dontCount {
		if skip.ArtifactID == artifact.ArtifactID {
			continue // Don't ignore the current artifact, or we won't get dependencies for it
		}
		ignore[skip.ArtifactID] = struct{}{}
	}

	result := buildplan.ArtifactChangeset{}

	deps := artifact.Dependencies(true, &ignore)
	for _, change := range changeset.Filter(buildplan.ArtifactAdded, buildplan.ArtifactUpdated) {
		for _, dep := range deps {
			if dep.ArtifactID == change.Artifact.ArtifactID {
				result = append(result, change)
			}
		}
	}

	return sliceutils.Unique(result)
}

func renderVersionChange(change structuredChange) string {
	if change.OldVersion == "" {
		return locale.Tr("upgrade_field_same", change.NewVersion)
	}
	old := change.OldVersion
	new := change.NewVersion
	if change.OldVersion == change.NewVersion {
		old = fmt.Sprintf("%s (%d)", old, change.OldRevision)
		new = fmt.Sprintf("%s (%d)", new, change.NewRevision)
	}
	if old == new {
		return locale.Tr("upgrade_field_same", change.NewVersion)
	}
	return locale.Tr("upgrade_field_change", change.OldVersion, change.NewVersion)
}

func renderLicenseChange(change structuredChange) string {
	from := change.OldLicenses
	to := change.NewLicenses
	if sliceutils.EqualValues(from, to) {
		return locale.Tr("upgrade_field_same", strings.Join(from, ", "))
	}
	return locale.Tr("upgrade_field_change", strings.Join(from, ", "), strings.Join(to, ", "))
}
