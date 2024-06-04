package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/runtime/internal/envdef"
	"github.com/go-openapi/strfmt"
)

const (
	depotFile = "depot.json"
)

type depotConfig struct {
	Deployments map[strfmt.UUID][]deployment `json:"deployments"`
}

type deployment struct {
	Type deploymentType `json:"type"`
	Path string         `json:"path"`
}

type deploymentType string

const (
	deploymentTypeLink deploymentType = "link"
	deploymentTypeCopy                = "copy"
)

type depot struct {
	config    depotConfig
	depotPath string
	artifacts map[strfmt.UUID]struct{}

	envDef *envdef.Collection
}

func newDepot(envDef *envdef.Collection) (*depot, error) {
	depotPath := filepath.Join(storage.CachePath(), depotName)

	if !fileutils.TargetExists(depotPath) {
		return &depot{}, nil
	}

	result := &depot{
		depotPath: depotPath,
		envDef:    envDef,
	}

	configFile := filepath.Join(depotPath, depotFile)
	if fileutils.TargetExists(configFile) {
		b, err := fileutils.ReadFile(configFile)
		if err != nil {
			return nil, errs.Wrap(err, "failed to read depot file")
		}
		if err := json.Unmarshal(b, &result.config); err != nil {
			return nil, errs.Wrap(err, "failed to unmarshal depot file")
		}
	}

	files, err := os.ReadDir(depotPath)
	if err != nil {
		return nil, errs.Wrap(err, "failed to read depot path")
	}

	result.artifacts = map[strfmt.UUID]struct{}{}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		if strfmt.IsUUID(file.Name()) {
			result.artifacts[strfmt.UUID(file.Name())] = struct{}{}
		}
	}

	return result, nil
}

func (d *depot) Exists(id strfmt.UUID) bool {
	_, ok := d.artifacts[id]
	return ok
}

func (d *depot) Path(id strfmt.UUID) string {
	return filepath.Join(d.depotPath, id.String())
}

// Put updates our depot with the given artifact ID. It will fail unless a folder by that artifact ID can be found in
// the depot.
// This allows us to write to the depot externally, and then call this function in order for the depot to ingest the
// necessary information. Writing externally is preferred because otherwise the depot would need a lot of specialized
// logic that ultimately don't really need to be a concern of the depot.
func (d *depot) Put(id strfmt.UUID) error {
	if !fileutils.TargetExists(d.Path(id)) {
		return errs.New("could not put %s, as dir does not exist: %s", id, d.Path(id))
	}
	d.artifacts[id] = struct{}{}
	return nil
}

// Deploy will take an artifact from the depot and deploy it to the target path.
// A deployment can be either a series of links or a copy of the files in question, depending on whether the artifact
// requires runtime specific transformations.
func (d *depot) Deploy(id strfmt.UUID, path string) error {
	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	// Collect artifact meta info
	var err error
	path, err = fileutils.ResolvePath(path)
	if err != nil {
		return errs.Wrap(err, "failed to resolve path")
	}

	artifactInfo, err := d.envDef.Load(d.Path(id))
	if err != nil {
		return errs.Wrap(err, "failed to get artifact info")
	}

	artifactInstallDir := filepath.Join(d.Path(id), artifactInfo.InstallDir())
	if !fileutils.DirExists(artifactInstallDir) {
		return errs.New("artifact installdir does not exist: %s", artifactInstallDir)
	}

	// Copy or link the artifact files, depending on whether the artifact in question relies on file transformations
	var deployType deploymentType
	if artifactInfo.NeedsTransforms() {
		if err := fileutils.CopyFiles(artifactInstallDir, path); err != nil {
			return errs.Wrap(err, "failed to copy artifact")
		}

		if err := artifactInfo.ApplyFileTransforms(path); err != nil {
			return errs.Wrap(err, "Could not apply env transforms")
		}

		deployType = deploymentTypeCopy
	} else {
		if err := fileutils.SmartLinkContents(artifactInstallDir, path); err != nil {
			return errs.Wrap(err, "failed to link artifact")
		}
		deployType = deploymentTypeLink
	}

	// Record deployment to config
	if _, ok := d.config.Deployments[id]; !ok {
		d.config.Deployments[id] = []deployment{}
	}
	d.config.Deployments[id] = append(d.config.Deployments[id], deployment{Type: deployType, Path: path})

	return nil
}

func (d *depot) Undeploy(id strfmt.UUID, path string) error {
	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	var err error
	path, err = fileutils.ResolvePath(path)
	if err != nil {
		return errs.Wrap(err, "failed to resolve path")
	}

	if err := d.envDef.Unload(d.Path(id)); err != nil {
		return errs.Wrap(err, "failed to get artifact info")
	}

	// Find record of our deployment
	deployments, ok := d.config.Deployments[id]
	if !ok {
		return errs.New("deployment for %s not found in depot", id)
	}
	deploy := sliceutils.Filter(deployments, func(d deployment) bool { return d.Path == path })
	if len(deploy) != 1 {
		return errs.New("no deployment found for %s in depot", path)
	}

	// Perform uninstall based on deployment type
	if deploy[0].Type == deploymentTypeCopy {
		if err := os.RemoveAll(path); err != nil {
			return errs.Wrap(err, "failed to remove artifact")
		}
	} else {
		if err := fileutils.SmartUnlinkContents(d.Path(id), path); err != nil {
			return errs.Wrap(err, "failed to unlink artifact")
		}
	}

	// Write changes to config
	d.config.Deployments[id] = sliceutils.Filter(d.config.Deployments[id], func(d deployment) bool { return d.Path != path })

	return nil
}

// Save will write config changes to disk (ie. links between depot artifacts and runtimes that use it).
// It will also delete any stale artifacts which are not used by any runtime.
func (d *depot) Save() error {
	// Delete artifacts that are no longer used
	for id := range d.artifacts {
		if deployments, ok := d.config.Deployments[id]; !ok || len(deployments) == 0 {
			if err := os.RemoveAll(d.Path(id)); err != nil {
				return errs.Wrap(err, "failed to remove stale artifact")
			}
		}
	}

	// Write config file changes to disk
	configFile := filepath.Join(d.depotPath, depotFile)
	b, err := json.Marshal(d.config)
	if err != nil {
		return errs.Wrap(err, "failed to marshal depot file")
	}
	if err := fileutils.WriteFile(configFile, b); err != nil {
		return errs.Wrap(err, "failed to write depot file")
	}
	return nil
}

func (d *depot) List(path string) map[strfmt.UUID]struct{} {
	path = fileutils.ResolvePathIfPossible(path)
	result := map[strfmt.UUID]struct{}{}
	for id, deploys := range d.config.Deployments {
		for _, p := range deploys {
			if fileutils.ResolvePathIfPossible(p.Path) == path {
				result[id] = struct{}{}
			}
		}
	}

	return result
}
