package commit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/archiver"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/brunoga/deep"
	"github.com/cespare/xxhash"
	"github.com/gowebpki/jcs"
)

const namespaceSuffixFiles = "files"
const cacheKeyFiles = "buildscript-file-%s"

type invalidDepsValueType struct{ error }

type invalidDepValueType struct{ error }

type invalidFeaturesValueType struct{ error }

type invalidFeatureValue struct{ error }

const keyHash = "hash_readonly"
const keyIngredientVersionID = "ingredient_version_id_readonly"
const keyRevision = "revision_readonly"

var readonlyKeys = []string{keyHash, keyIngredientVersionID, keyRevision}

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

	resolvedIngredient, err := i.getCached(hash)
	if err != nil {
		return errs.Wrap(err, "Could not check if ingredient call is cached")
	}
	if resolvedIngredient == nil {
		resolvedIngredient, err = i.createIngredient(hash, hashedFiles)
		if err != nil {
			return errs.Wrap(err, "Could not create ingredient")
		}
		// Bump timestamp, because otherwise the new ingredient will be unusable
		latest, err := model.FetchLatestRevisionTimeStamp(nil)
		if err != nil {
			return errs.Wrap(err, "Unable to determine latest revision time")
		}
		i.script.SetAtTime(latest, true)
	}

	// Add/update arguments on the buildscript ingredient function call
	// ingredient_version_id and revision are required for the server to lookup the ingredient without violating
	// reproducibility.
	// hash is used to uniquely identify the ingredient and is used to cache the ingredient, it is only evaluated on the client.
	i.funcCall.SetArgument(keyIngredientVersionID, buildscript.Value(resolvedIngredient.VersionID))
	i.funcCall.SetArgument(keyRevision, buildscript.Value(resolvedIngredient.Revision))
	i.funcCall.SetArgument(keyHash, buildscript.Value(hash))
	i.setCached(hash, resolvedIngredient)

	return nil
}

func (i *IngredientCall) createIngredient(hash string, hashedFiles []*graph.GlobFileResult) (*resolvedIngredientData, error) {
	// Create tar.gz with all the references files for this ingredient
	files := []string{}
	for _, f := range hashedFiles {
		files = append(files, f.Path)
	}

	tmpFile := fileutils.TempFilePath("", fmt.Sprintf("bs-hash-%s.tar.gz", hash))
	if err := archiver.CreateTgz(tmpFile, archiver.FilesWithCommonParent(files...)); err != nil {
		return nil, errs.Wrap(err, "Could not create tar.gz")
	}
	defer os.Remove(tmpFile)

	// Parse buildscript dependencies
	deps, err := i.resolveDependencies()
	if err != nil {
		return nil, errs.Wrap(err, "Could not resolve dependencies")
	}

	// Parse buildscript features
	features, err := i.resolveFeatures()
	if err != nil {
		return nil, errs.Wrap(err, "Could not resolve features")
	}

	// Publish ingredient
	bpm := buildplanner.NewBuildPlannerModel(i.prime.Auth(), i.prime.SvcModel())
	pr, err := bpm.Publish(request.PublishVariables{
		Name:         hash,
		Description:  "buildscript ingredient",
		Namespace:    i.ns.String(),
		Version:      hash,
		Dependencies: deps,
		Features:     features,
	}, tmpFile)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create publish request")
	}

	return &resolvedIngredientData{
		VersionID: pr.IngredientVersionID,
		Revision:  pr.Revision,
	}, nil
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
	for _, k := range readonlyKeys {
		fcc.UnsetArgument(k)
	}

	fcb, err := json.Marshal(fcc)
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal function call")
	}
	// Go's JSON implementation does not produce canonical output, so we need to utilize additional tooling to ensure
	// the hash we create is based on canonical data.
	if fcb, err = jcs.Transform(fcb); err != nil {
		return "", errs.Wrap(err, "Could not transform json blob to canonical json")
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
				VersionRequirements: model.VersionRequirementsToString(req.VersionRequirement, true),
				Type:                typ,
			},
			[]request.Dependency{},
		})
	}

	return deps, nil
}

// resolveFeatures turns ingredient features into the appropriate types used by our models
func (i *IngredientCall) resolveFeatures() ([]request.PublishVariableFeature, error) {
	features := []request.PublishVariableFeature{}
	bsFeatures := i.funcCall.Argument("features")
	if bsFeatures == nil {
		return features, nil
	}

	bsFeaturesSlice, ok := bsFeatures.([]string)
	if !ok {
		return nil, invalidFeaturesValueType{fmt.Errorf("features argument is not an []string: %v (%T)", bsFeatures, bsFeatures)}
	}

	for _, feature := range bsFeaturesSlice {
		resolvedFeature, err := parseFeature(feature)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse feature")
		}
		features = append(features, resolvedFeature)
	}

	return features, nil
}

// parseFeature will parse a feature string into a request.PublishVariableFeature
// features need to be formatted as `namespace/name@version`
func parseFeature(f string) (request.PublishVariableFeature, error) {
	p := request.PublishVariableFeature{}
	if !strings.Contains(f, "/") {
		return p, &invalidFeatureValue{fmt.Errorf("feature '%s' is missing a namespace, it should be formatted as namespace/name@version", f)}
	}
	if !strings.Contains(f, "@") {
		return p, &invalidFeatureValue{fmt.Errorf("feature '%s' is missing a version, it should be formatted as namespace/name@version", f)}
	}
	v := strings.Split(f, "@")
	p.Version = strings.TrimSpace(v[1])
	v = strings.Split(v[0], "/")
	p.Namespace = strings.TrimSpace(strings.Join(v[0:len(v)-1], "/"))
	p.Name = strings.TrimSpace(v[len(v)-1])
	return p, nil
}

type resolvedIngredientData struct {
	VersionID string
	Revision  int
}

// getCached checks against our local cache to see if we've already handled this file hash, and if no local cache
// exists checks against the platform ingredient API.
func (i *IngredientCall) getCached(hash string) (*resolvedIngredientData, error) {
	cacheValue, err := i.prime.SvcModel().GetCache(fmt.Sprintf(cacheKeyFiles, hash))
	if err != nil {
		return nil, errs.Wrap(err, "Could not get build script cache")
	}
	if cacheValue != "" {
		resolvedIngredient := &resolvedIngredientData{}
		err := json.Unmarshal([]byte(cacheValue), resolvedIngredient)
		if err != nil {
			return nil, errs.Wrap(err, "Could not unmarshal cached ingredient")
		}
		// Ingredient already exists
		return resolvedIngredient, nil
	}

	// Check against API to see if we've already published this file hash
	ingredients, err := model.SearchIngredientsStrict(i.ns.String(), hash, true, false, i.script.AtTime(), i.prime.Auth())
	if err != nil && !errors.As(err, ptr.To(&model.ErrSearch404{})) {
		return nil, errs.Wrap(err, "Could not search ingredients")
	}
	if len(ingredients) > 0 {
		// Ingredient already exists
		return &resolvedIngredientData{
			VersionID: string(*ingredients[0].LatestVersion.IngredientVersionID),
			Revision:  int(*ingredients[0].LatestVersion.Revision),
		}, nil
	}

	return nil, nil // If we made it this far it means we did not find any existing cache entry; so it's dirty
}

// Update our local cache saying we've handled this hash, allowing for faster cache checks than using the platform api
func (i *IngredientCall) setCached(hash string, resolvedIngredient *resolvedIngredientData) {
	b, err := json.Marshal(resolvedIngredient)
	if err != nil {
		// Shouldn't happen, but at least we're logging it if it does
		logging.Error("Could not marshal cached ingredient: %s", errs.JoinMessage(err))
	}
	err = i.prime.SvcModel().SetCache(fmt.Sprintf(cacheKeyFiles, hash), string(b), time.Hour*24*7)
	if err != nil {
		logging.Error("Could not set build script cache: %s", errs.JoinMessage(err))
	}
}
