package deptree

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type DeptreeByIngredients struct {
	prime primeable
}

func NewByIngredients(prime primeable) *DeptreeByIngredients {
	return &DeptreeByIngredients{
		prime: prime,
	}
}

type IngredientParams struct {
	Namespace  *project.Namespaced
	CommitID   string
	Req        string
	LevelLimit int
}

func (d *DeptreeByIngredients) Run(params IngredientParams) error {
	logging.Debug("Execute DeptreeByIngredients")

	proj := d.prime.Project()
	if proj == nil {
		return rationalize.ErrNoProject
	}

	ns, err := resolveNamespace(params.Namespace, params.CommitID, d.prime)
	if err != nil {
		return errs.Wrap(err, "Could not resolve namespace")
	}

	bpm := buildplanner.NewBuildPlannerModel(d.prime.Auth(), d.prime.SvcModel())
	commit, err := bpm.FetchCommit(*ns.CommitID, ns.Owner, ns.Project, nil)
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr and time for provided commit")
	}

	bp := commit.BuildPlan()

	levelLimit := params.LevelLimit
	if levelLimit == 0 {
		levelLimit = 10
	}

	ingredients := bp.RequestedIngredients()
	common := ingredients.CommonRuntimeDependencies().ToIDMap()

	// Ensure languages are listed first, because they tend to themselves be dependencies
	sort.Slice(ingredients, func(i, j int) bool { return ingredients[i].Namespace == model.NamespaceLanguage.String() })

	if params.Req != "" {
		ingredients = sliceutils.Filter(ingredients, func(i *buildplan.Ingredient) bool {
			return i.Name == params.Req
		})
	}

	d.printIngredients(
		ingredients,
		common,
		0,
		levelLimit,
		make(map[strfmt.UUID]struct{}),
	)

	return nil
}

const indentValue = "  "

func (d *DeptreeByIngredients) printIngredients(
	is buildplan.Ingredients,
	common buildplan.IngredientIDMap,
	level int,
	levelLimit int,
	seen map[strfmt.UUID]struct{},
) {
	indent := strings.Repeat(indentValue, level)
	if level == levelLimit {
		d.prime.Output().Print(indent + indentValue + "[ORANGE]Recursion limit reached[/RESET]")
		return
	}
	count := 0
	for _, i := range is {
		count = count + 1

		color := "CYAN"
		if _, ok := common[i.IngredientID]; ok {
			color = "YELLOW"
		}
		d.prime.Output().Print(fmt.Sprintf("%s%d. [%s]%s/%s[/RESET] ([DISABLED]%s[/RESET])",
			indent, count, color, i.Namespace, i.Name, i.IngredientID))

		if _, ok := seen[i.IngredientID]; ok {
			d.prime.Output().Print(fmt.Sprintf(
				indent + indentValue + indentValue + "[DISABLED]Already listed[/RESET]",
			))
			continue
		}
		seen[i.IngredientID] = struct{}{}

		d.printIngredients(
			i.RuntimeDependencies(false),
			common,
			level+1,
			levelLimit,
			seen,
		)
	}
}
