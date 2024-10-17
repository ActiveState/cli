package commit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/brunoga/deep"
	"github.com/cespare/xxhash"
	"github.com/mholt/archiver/v3"
)

const namespaceSuffixFiles = "files"
const cacheKeyFiles = "buildscript-file-%s"

type invalidDepsValueType struct{ error }

type invalidDepValueType struct{ error }

// IngredientCall is used to evaluate ingredient() function calls and publishes the ingredient in question if it is not
// already published.
type IngredientCall struct {
	prime    primeable
	script   *buildscript.BuildScript
	funcCall *buildscript.FuncCall
	ns       model.Namespace
}

func NewIngredientCall(
	prime primeable,
	script *buildscript.BuildScript,
	funcCall *buildscript.FuncCall,
) *IngredientCall {
	return &IngredientCall{
		prime:    prime,
		script:   script,
		funcCall: funcCall,
		ns:       model.NewNamespaceOrg(prime.Project().Owner(), namespaceSuffixFiles),
	}
}

// Resolve will check if the ingredient call refers to an existing ingredient, and if not will create it and update
// the buildscript accordingly.
func (i *IngredientCall) Resolve() error {
	hash, hashedFiles, err := i.calculateHash()
	if err != nil {
		return errs.Wrap(err, "Could not hash ingredient call")
	}

	cached, err := i.isCached(hash)
	if err != nil {
		return errs.Wrap(err, "Could not check if ingredient call is cached")
	}
	if cached {
		// Ingredient already exists, nothing to do
		return nil
	}

	// Creative tar.gz with all the references files for this ingredient
	files := []string{}
	for _, f := range hashedFiles {
		files = append(files, f.Path)
	}
	tmpFile := fileutils.TempFilePath("", fmt.Sprintf("bs-hash-%s.tar.gz", hash))
	if err := archiver.Archive(files, tmpFile); err != nil {
		return errs.Wrap(err, "Could not archive files")
	}
	defer os.Remove(tmpFile)

	// Parse buildscript dependencies
	deps, err := i.resolveDependencies()
	if err != nil {
		return errs.Wrap(err, "Could not resolve dependencies")
	}

	// Publish ingredient
	bpm := buildplanner.NewBuildPlannerModel(i.prime.Auth(), i.prime.SvcModel())
	_, err = bpm.Publish(request.PublishVariables{
		Name:         hash,
		Namespace:    i.ns.String(),
		Dependencies: deps,
	}, tmpFile)
	if err != nil {
		return errs.Wrap(err, "Could not create publish request")
	}

	// Add/update hash argument on the buildscript ingredient function call
	i.funcCall.SetArgument("hash", buildscript.Value(hash))
	i.setCached(hash)

	return nil
}

// calculateHash will calculate a hash based on the files references in the ingredient as well as the ingredient
// rule itself. The ingredient is considered dirty when either the files or the rule itself has changed.
func (i *IngredientCall) calculateHash() (string, []*graph.GlobFileResult, error) {
	src := i.funcCall.Argument("src")
	patterns, ok := src.([]string)
	if !ok {
		return "", nil, errors.New("src argument is not a []string")
	}
	hashed, err := i.prime.SvcModel().HashGlobs(i.prime.Project().Dir(), patterns)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not hash globs")
	}

	hash, err := hashFuncCall(i.funcCall, hashed.Hash)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not hash function call")
	}

	return hash, hashed.Files, nil
}

// hashFuncCall will calculate the individual hash of the ingredient function call itself.
// The hash argument is excluded from this calculation.
func hashFuncCall(fc *buildscript.FuncCall, seed string) (string, error) {
	// We clone the function call here because the (potentially old) hash itself should not be used to calculate the hash
	// and unsetting it should not propagate beyond the context of this function.
	fcc, err := deep.Copy(fc)
	if err != nil {
		return "", errs.Wrap(err, "Could not copy function call")
	}
	fcc.UnsetArgument("hash")

	fcb, err := json.Marshal(fcc)
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal function call")
	}
	hasher := xxhash.New()
	hasher.Write([]byte(seed))
	hasher.Write(fcb)
	hash := fmt.Sprintf("%016x", hasher.Sum64())
	return hash, nil
}

// resolveDependencies iterates over the different dependency arguments the ingredient function supports and resolves
// them into the appropriate types used by our models.
func (i *IngredientCall) resolveDependencies() ([]request.PublishVariableDep, error) {
	result := []request.PublishVariableDep{}
	for key, typ := range map[string]request.DependencyType{
		"runtime_deps": request.DependencyTypeRuntime,
		"build_deps":   request.DependencyTypeBuild,
		"test_deps":    request.DependencyTypeTest,
	} {
		deps, err := i.resolveDependenciesByKey(key, typ)
		if err != nil {
			return nil, errs.Wrap(err, "Could not resolve %s", key)
		}
		result = append(result, deps...)
	}

	return result, nil
}

// resolveDependenciesByKey turns ingredient dependencies into the appropriate types used by our models
func (i *IngredientCall) resolveDependenciesByKey(key string, typ request.DependencyType) ([]request.PublishVariableDep, error) {
	deps := []request.PublishVariableDep{}
	bsDeps := i.funcCall.Argument(key)
	if bsDeps == nil {
		return deps, nil
	}

	bsDepSlice, ok := bsDeps.([]any)
	if !ok {
		return nil, invalidDepsValueType{fmt.Errorf("deps argument is not a []any: %v (%T)", bsDeps, bsDeps)}
	}

	for _, dep := range bsDepSlice {
		req, ok := dep.(buildscript.DependencyRequirement)
		if !ok {
			return nil, invalidDepValueType{fmt.Errorf("dep argument is not a Req(): %v (%T)", dep, dep)}
		}
		deps = append(deps, request.PublishVariableDep{
			request.Dependency{
				Name:                req.Name,
				Namespace:           req.Namespace,
				VersionRequirements: model.BuildPlannerVersionConstraintsToString(req.VersionRequirement),
				Type:                typ,
			},
			[]request.Dependency{},
		})
	}

	return deps, nil
}

// isCached checks against our local cache to see if we've already handled this file hash, and if no local cache
// exists checks against the platform ingredient API.
func (i *IngredientCall) isCached(hash string) (bool, error) {
	cacheValue, err := i.prime.SvcModel().GetCache(fmt.Sprintf(cacheKeyFiles, hash))
	if err != nil {
		return false, errs.Wrap(err, "Could not get build script cache")
	}
	if cacheValue != "" {
		// Ingredient already exists
		return true, nil
	}

	// Check against API to see if we've already published this file hash
	ingredients, err := model.SearchIngredientsStrict(i.ns.String(), hash, true, false, i.script.AtTime(), i.prime.Auth())
	if err != nil {
		return false, errs.Wrap(err, "Could not search ingredients")
	}
	if len(ingredients) > 0 {
		// Ingredient already exists
		return true, nil
	}

	return false, nil // If we made it this far it means we did not find any existing cache entry; so it's dirty
}

// Update our local cache saying we've handled this hash, allowing for faster cache checks than using the platform api
func (i *IngredientCall) setCached(hash string) {
	err := i.prime.SvcModel().SetCache(fmt.Sprintf(cacheKeyFiles, hash), hash, time.Hour*24*7)
	if err != nil {
		logging.Warning("Could not set build script cache: %s", errs.JoinMessage(err))
	}
}
