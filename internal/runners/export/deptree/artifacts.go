package deptree

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
)

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Configurer
	primer.Projecter
	primer.Analyticer
	primer.SvcModeler
	primer.CheckoutInfoer
}

type ArtifactParams struct {
	Namespace  *project.Namespaced
	CommitID   string
	Req        string
	PlatformID string
	LevelLimit int
}

type DeptreeByArtifacts struct {
	prime primeable
}

func NewByArtifacts(prime primeable) *DeptreeByArtifacts {
	return &DeptreeByArtifacts{
		prime: prime,
	}
}

func (d *DeptreeByArtifacts) Run(params ArtifactParams) error {
	logging.Debug("Execute DepTree")

	out := d.prime.Output()
	proj := d.prime.Project()
	if proj == nil {
		return rationalize.ErrNoProject
	}

	ns, err := resolveNamespace(params.Namespace, params.CommitID, d.prime)
	if err != nil {
		return errs.Wrap(err, "Could not resolve namespace")
	}

	bpm := buildplanner.NewBuildPlannerModel(d.prime.Auth(), d.prime.SvcModel())
	commit, err := bpm.FetchCommit(*ns.CommitID, ns.Owner, ns.Project, proj.BranchName(), nil)
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr and time for provided commit")
	}

	bp := commit.BuildPlan()

	platformID := strfmt.UUID(params.PlatformID)
	if platformID == "" {
		platformID, err = model.FilterCurrentPlatform(sysinfo.OS().String(), bp.Platforms(), "")
		if err != nil {
			return errs.Wrap(err, "Could not get platform ID")
		}
	}

	levelLimit := params.LevelLimit
	if levelLimit == 0 {
		levelLimit = 10
	}

	ingredients := bp.RequestedIngredients()
	for _, ingredient := range ingredients {
		if params.Req != "" && ingredient.Name != params.Req {
			continue
		}
		out.Print(fmt.Sprintf("• [ACTIONABLE]%s/%s[/RESET] ([DISABLED]%s[/RESET])", ingredient.Namespace, ingredient.Name, ingredient.IngredientID))
		d.printArtifacts(
			nil,
			ingredient.Artifacts.Filter(
				buildplan.FilterPlatformArtifacts(platformID),
			),
			platformID,
			1,
			levelLimit,
		)
	}

	return nil
}

func (d *DeptreeByArtifacts) printArtifacts(
	parents []*buildplan.Artifact,
	as buildplan.Artifacts,
	platformID strfmt.UUID,
	level int,
	levelLimit int) {
	indent := strings.Repeat("  ", level)
	if level == levelLimit {
		d.prime.Output().Print(indent + indentValue + "[ORANGE]Recursion limit reached[/RESET]")
		return
	}
	count := 0
	for _, a := range as {
		if len(sliceutils.Filter(parents, func(p *buildplan.Artifact) bool { return p.ArtifactID == a.ArtifactID })) != 0 {
			d.prime.Output().Print(fmt.Sprintf("%s • Recurse to [CYAN]%s[/RESET] ([DISABLED]%s[/RESET])", indent, a.DisplayName, a.ArtifactID))
			continue
		}
		depTypes := []string{}
		if a.IsRuntimeDependency {
			depTypes = append(depTypes, "[GREEN]Runtime[/RESET]")
		}
		if a.IsBuildtimeDependency {
			depTypes = append(depTypes, "[ORANGE]Buildtime[/RESET]")
		}
		mime := ""
		if !buildplanner.IsStateToolArtifact(a.MimeType) {
			mime = fmt.Sprintf(" ([DISABLED]%s[/RESET])", a.MimeType)
		}
		count = count + 1
		d.prime.Output().Print(fmt.Sprintf("%s%d. [CYAN]%s[/RESET] [%s] ([DISABLED]%s[/RESET]) %s", indent, count, a.DisplayName, strings.Join(depTypes, "|"), a.ArtifactID, mime))
		d.printArtifacts(
			append(parents, a),
			a.Dependencies(false, nil).Filter(
				buildplan.FilterPlatformArtifacts(platformID),
			),
			platformID,
			level+1,
			levelLimit,
		)
	}
}
