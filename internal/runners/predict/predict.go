package predict

import (
	"fmt"
	"math"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/predict/models"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"github.com/google/martian/log"
)

type Predict struct {
	pj *project.Project
}

type primeable interface {
	primer.Projecter
}

type PredictParams struct {
	Names []string
}

type VersionedOrder struct {
	Order   *inventory_models.Order
	Version string
}

func New(prime primeable) *Predict {
	return &Predict{prime.Project()}
}

func splitNameAndVersion(input string) (string, string) {
	nameArg := strings.Split(input, "@")
	name := nameArg[0]
	version := ""
	if len(nameArg) == 2 {
		version = nameArg[1]
	}

	return name, version
}

func (a *Predict) Run(params PredictParams) error {
	if a.pj == nil {
		return locale.NewInputError("err_no_project")
	}

	language, err := model.LanguageForCommit(a.pj.CommitUUID())
	if err != nil {
		return locale.WrapError(err, "err_fetch_languages")
	}

	packageNameSpace := model.NewNamespacePkgOrBundle(language, model.NamespacePackage)
	parentCommitID := a.pj.CommitUUID()

	var preds []*prediction
	for _, name := range params.Names {
		packageName, packageVersion := splitNameAndVersion(name)
		operation := model.OperationAdded

		// add package
		commitID, err := model.CommitPackage(
			parentCommitID, operation,
			packageName, packageNameSpace.String(), packageVersion,
			machineid.UniqID())
		if err != nil {
			return errs.Wrap(err, "Could not create commit with new version")
		}
		log.Debugf("added package commit id: %s\n", commitID)

		// create new order
		order, err := model.CommitToOrder(commitID, a.pj.Owner(), a.pj.Name())
		if err != nil {
			return errs.Wrap(err, "Could not retrieve order")
		}
		log.Debugf("added package order: %s\n", order.OrderID)

		// get version numbers
		ingVers, err := model.SearchIngredientsStrict(packageNameSpace, packageName)
		if err != nil {
			return errs.Wrap(err, "Could not retrieve ingredient versions.")
		}
		log.Debugf("found %d versions for ingredient\n", len(ingVers))

		// create versioned orders
		eqOp := inventory_models.RequirementComparatorEq
		for _, iv := range ingVers {
			newOrder := order
			for i, rc := range newOrder.Requirements {
				if rc.IngredientVersionID == *iv.LatestVersion.IngredientVersionID {
					newRc := rc
					newRc.VersionRequirements = []*inventory_models.VersionRequirement{
						{Comparator: &eqOp, Version: &iv.Version},
					}
					newOrder.Requirements[i] = newRc
				}
			}
			vo := &VersionedOrder{order, fmt.Sprintf("%s %s", name, iv.Version)}

			recipe, err := model.FetchRecipeForOrder(commitID, vo.Order, a.pj.Owner(), a.pj.Name(), &model.HostPlatform)
			if err != nil {
				return errs.Wrap(err, "Could not fetch recipe for order")
			}
			log.Debugf("got recipe %s\n", recipe.RecipeID)

			dag, err := models.ParseRecipe(recipe)
			if err != nil {
				return errs.Wrap(err, "Failed to parse the build dag for recipe: %w", err)
			}

			pred := parseDag(dag)
			pred.PackageVersion = vo.Version

			fmt.Printf("Prediction for %s: %s\n", pred.PackageVersion, pred.String())
			if len(pred.failed) > 0 {
				for _, f := range pred.failed {
					fmt.Printf("Failed due to artifact %s: %s\n", f.fullName, f.log)
				}
			}

			preds = append(preds, pred)
		}
	}

	return nil
}

func walkTheDag(a *models.Artifact, flatGraph map[string]*models.Artifact, f func(b *models.Artifact)) {
	if len(a.BuildDependencies) == 0 {
		return
	}

	for _, bd := range a.BuildDependencies {
		if _, ok := flatGraph[bd.ArtifactID.String()]; ok {
			continue
		}
		flatGraph[bd.ArtifactID.String()] = bd

		f(bd)
		walkTheDag(bd, flatGraph, f)
	}
}

func parseDag(dag *models.RecipeBuildDAG) *prediction {
	flat := make(map[string]*models.Artifact)
	var unresolved []string
	var failed []failedPrediction

	collector := func(b *models.Artifact) {
		c, err := headchef.InitClient().GetArtifactStatus(strfmt.UUID(b.ArtifactID.String()))
		if err != nil {
			unresolved = append(unresolved, b.Ingredient.FullName())
			return
		}
		// fmt.Printf("%s %s %s\n", b.Ingredient.FullName(), c.ArtifactID.String(), *c.BuildState)
		if *c.BuildState == "failed" {
			failed = append(failed, failedPrediction{b.Ingredient.FullName(), c.LogURI.String()})
			return
		}

		if *c.BuildState == "succeeded" {
			return
		}

		unresolved = append(unresolved, b.Ingredient.FullName())
	}

	walkTheDag(dag.TerminalArtifact, flat, collector)

	return &prediction{
		numDependencies: len(flat),
		successful:      len(unresolved) == 0 && len(failed) == 0,
		unresolved:      unresolved,
		failed:          failed,
	}
}

type failedPrediction struct {
	fullName string
	log      string
}

type prediction struct {
	PackageVersion  string
	numDependencies int
	successful      bool
	unresolved      []string
	failed          []failedPrediction
}

func (pr *prediction) String() string {
	if pr.successful {
		return "✓ 100%"
	}
	if len(pr.failed) > 0 {
		return "❌ 0%"
	}

	prob := math.Pow(0.99, float64(len(pr.unresolved)))
	return fmt.Sprintf("%2.1f%% %d un-built dependencies", prob*100, len(pr.unresolved))
}
