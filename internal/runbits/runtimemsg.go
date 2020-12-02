package runbits

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/vbauerster/mpb/v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type requestedRequirement struct {
	name      string
	namespace model.Namespace
}

type RuntimeMessageHandler struct {
	out  output.Outputer
	bpg  *progress.Progress
	bbar *progress.TotalBar

	requirement *requestedRequirement
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out, nil, nil, nil}
}

// SetChangeSummaryFunc sets a function that is called after the build recipe is known and can display a summary of changes that happened to the build
func (r *RuntimeMessageHandler) SetRequirement(name string, namespace model.Namespace) {
	r.requirement = &requestedRequirement{name, namespace}
}

func (r *RuntimeMessageHandler) DownloadStarting() {
	r.out.Notice(output.Heading(locale.T("downloading_artifacts")))
}

func (r *RuntimeMessageHandler) InstallStarting() {
	r.out.Notice(output.Heading(locale.T("installing_artifacts")))
}

func (r *RuntimeMessageHandler) ChangeSummary(directDeps map[strfmt.UUID][]strfmt.UUID, recursiveDeps map[strfmt.UUID][]strfmt.UUID, ingredientMap map[strfmt.UUID]*inventory_models.ResolvedIngredient) {
	if r.requirement == nil {
		return
	}

	var matchedIngredient *inventory_models.ResolvedIngredient
	for _, ingredient := range ingredientMap {
		if matchedIngredient != nil {
			break
		}
		for _, req := range ingredient.ResolvedRequirements {
			if req.Feature != nil && *req.Feature == r.requirement.name &&
				req.Namespace != nil && *req.Namespace == r.requirement.namespace.String() {
				matchedIngredient = ingredient
				break
			}
		}
	}

	if matchedIngredient == nil {
		logging.Error("Could not find requirement in resulting recipe: %s (%s)", r.requirement.name, r.requirement.namespace)
		return
	}

	depsForReq, ok := directDeps[*matchedIngredient.IngredientVersion.IngredientVersionID]
	if !ok {
		logging.Error("Could not find deps for supplied requirement: %s (%s)", r.requirement.name, r.requirement.namespace)
		return
	}

	countDirect := len(depsForReq)
	countTotal := len(ingredientMap)

	r.out.Notice("")
	r.out.Notice(locale.Tl(
		"changesummary_title",
		"[NOTICE]{{.V0}}[/RESET] includes {{.V1}} dependencies, for a combined total of {{.V2}} dependencies.",
		*matchedIngredient.Ingredient.Name, strconv.Itoa(countDirect), strconv.Itoa(countTotal),
	))
	for i, dep := range depsForReq {
		depMapping, ok := ingredientMap[dep]
		if !ok {
			logging.Error("Could not find dependency %s in ingredientMap", dep)
			continue
		}
		var depCount string
		recDeps, ok := recursiveDeps[dep]
		if !ok {
			logging.Error("Could not find recursive dependency of ingredient %s", dep)
		}
		if len(recDeps) > 0 {
			depCount = locale.Tl("ingredient_dependency_count", " ({{.V0}} dependencies)", strconv.Itoa(len(recDeps)))
		}
		prefix := "├─"
		if i == len(depsForReq)-1 {
			prefix = "└─"
		}
		r.out.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s%s", prefix, *depMapping.Ingredient.Name, depCount))
	}
}

func (r *RuntimeMessageHandler) BuildStarting(totalArtifacts int) {
	logging.Debug("BuildStarting")
	if r.bpg != nil || r.bbar != nil {
		logging.Error("BuildStarting: progress has already initialized")
		return
	}

	progressOut := os.Stderr
	if strings.ToLower(os.Getenv(constants.NonInteractive)) == "true" {
		progressOut = nil
	}

	r.bpg = progress.New(mpb.WithOutput(progressOut))
	r.bbar = r.bpg.AddTotalBar(locale.Tl("building_remotely", "Building Remotely"), totalArtifacts)
}

func (r *RuntimeMessageHandler) BuildFinished() {
	if r.bpg == nil || r.bbar == nil {
		logging.Error("BuildFinished: progressbar is nil")
		return
	}

	logging.Debug("BuildFinished")
	if !r.bbar.Completed() {
		r.bpg.Cancel()
	}
	r.bpg.Close()
}

func (r *RuntimeMessageHandler) ArtifactBuildStarting(artifactName string) {
	logging.Debug("ArtifactBuildStarting: %s", artifactName)
}

func (r *RuntimeMessageHandler) ArtifactBuildCached(artifactName string) {
	logging.Debug("ArtifactBuildCached: %s", artifactName)
}

func (r *RuntimeMessageHandler) ArtifactBuildCompleted(artifactName string, number, total int) {
	if r.bpg == nil || r.bbar == nil {
		logging.Error("ArtifactBuildCompleted: progressbar is nil")
		return
	}

	logging.Debug("ArtifactBuildCompleted: %s", artifactName)
	r.bbar.Increment()
}

func (r *RuntimeMessageHandler) ArtifactBuildFailed(artifactName string, errorMsg string) {
	logging.Debug("ArtifactBuildFailed: %s: %s", artifactName, errorMsg)
}
