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

func (c *Commit) resolveIngredientCall(script *buildscript.BuildScript, fc *buildscript.FuncCall) error {
	hash, hashedFiles, err := c.hashIngredientCall(fc)
	if err != nil {
		return errs.Wrap(err, "Could not hash ingredient call")
	}

	ns := model.NewNamespaceOrg(c.prime.Project().Owner(), namespaceSuffixFiles)

	cached, err := c.isIngredientCallCached(script.AtTime(), ns, hash)
	if err != nil {
		return errs.Wrap(err, "Could not check if ingredient call is cached")
	}
	if cached {
		return nil
	}

	files := []string{}
	for _, f := range hashedFiles {
		files = append(files, f.Path)
	}
	tmpFile := fileutils.TempFilePath("", fmt.Sprintf("bs-hash-%s.tar.gz", hash))
	if err := archiver.Archive(files, tmpFile); err != nil {
		return errs.Wrap(err, "Could not archive files")
	}
	defer os.Remove(tmpFile)

	deps, err := c.resolveDependencies(fc)
	if err != nil {
		return errs.Wrap(err, "Could not resolve dependencies")
	}

	// Publish ingredient
	bpm := buildplanner.NewBuildPlannerModel(c.prime.Auth(), c.prime.SvcModel())
	_, err = bpm.Publish(request.PublishVariables{
		Name:         hash,
		Namespace:    ns.String(),
		Dependencies: deps,
	}, tmpFile)
	if err != nil {
		return errs.Wrap(err, "Could not create publish request")
	}

	// Add/update hash argument on the buildscript ingredient function call
	fc.SetArgument("hash", buildscript.Value(hash))
	c.setIngredientCallCached(hash)

	return nil
}

func (c *Commit) hashIngredientCall(fc *buildscript.FuncCall) (string, []*graph.GlobFileResult, error) {
	src := fc.Argument("src")
	patterns, ok := src.([]string)
	if !ok {
		return "", nil, errors.New("src argument is not a []string")
	}
	hashed, err := c.prime.SvcModel().HashGlobs(c.prime.Project().Dir(), patterns)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not hash globs")
	}

	// Combine file hash with function call hash
	fcc, err := deep.Copy(fc)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not copy function call")
	}
	fcc.UnsetArgument("hash") // The (potentially old) hash itself should not be used to calculate the hash

	fcb, err := json.Marshal(fcc)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not marshal function call")
	}
	hasher := xxhash.New()
	hasher.Write([]byte(hashed.Hash))
	hasher.Write(fcb)
	hash := fmt.Sprintf("%016x", hasher.Sum64())

	return hash, hashed.Files, nil
}

type invalidDepsValueType struct {
	error
}

type invalidDepValueType struct {
	error
}

func (c *Commit) resolveDependencies(fc *buildscript.FuncCall) ([]request.PublishVariableDep, error) {
	deps := []request.PublishVariableDep{}
	bsDeps := fc.Argument("deps")
	if bsDeps != nil {
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
			},
			[]request.Dependency{},
		})
	}

	return deps, nil
}

func (c *Commit) isIngredientCallCached(atTime *time.Time, ns model.Namespace, hash string) (bool, error) {
	// Check against our local cache to see if we've already handled this file hash
	// Technically we don't need this because the SearchIngredients call below already verifies this, but searching
	// ingredients is slow, and local cache is FAST.
	cacheValue, err := c.prime.SvcModel().GetCache(fmt.Sprintf(cacheKeyFiles, hash))
	if err != nil {
		return false, errs.Wrap(err, "Could not get build script cache")
	}
	if cacheValue != "" {
		// Ingredient already exists
		return true, nil
	}

	// Check against API to see if we've already published this file hash
	ingredients, err := model.SearchIngredientsStrict(ns.String(), hash, true, false, atTime, c.prime.Auth())
	if err != nil {
		return false, errs.Wrap(err, "Could not search ingredients")
	}
	if len(ingredients) > 0 {
		// Ingredient already exists
		return true, nil
	}

	return false, nil // If we made it this far it means we did not find any existing cache entry; so it's dirty
}

func (c *Commit) setIngredientCallCached(hash string) {
	err := c.prime.SvcModel().SetCache(fmt.Sprintf(cacheKeyFiles, hash), hash, time.Hour*24*7)
	if err != nil {
		logging.Warning("Could not set build script cache: %s", errs.JoinMessage(err))
	}
}
