package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/smartlink"
)

const (
	depotFile = "depot.json"
)

type depotConfig struct {
	Deployments map[strfmt.UUID][]deployment `json:"deployments"`
}

type deployment struct {
	Type deploymentType `json:"type"`

	// Path is unused at the moment: in the future we should record the exact paths that were deployed, so we can track
	// file ownership when multiple artifacts deploy the same file.
	// I've left this in so it's clear why we're employing a struct here rather than just have each deployment contain
	// the deploymentType as the direct value. That would make it more difficult to update this logic later due to
	// backward compatibility.
	// Path string         `json:"path"`
}

type deploymentType string

const (
	deploymentTypeLink deploymentType = "link"
	deploymentTypeCopy                = "copy"
)

type depot struct {
	config     depotConfig
	depotPath  string
	targetPath string
	artifacts  map[strfmt.UUID]struct{}
}

func newDepot(targetPath string) (*depot, error) {
	if fileutils.TargetExists(targetPath) && !fileutils.IsDir(targetPath) {
		return nil, errors.New(fmt.Sprintf("target path must be a directory: %s", targetPath))
	}

	depotPath := filepath.Join(storage.CachePath(), depotName)
	configFile := filepath.Join(targetPath, configDir, depotFile)

	result := &depot{
		config: depotConfig{
			Deployments: map[strfmt.UUID][]deployment{},
		},
		depotPath:  depotPath,
		targetPath: targetPath,
		artifacts:  map[strfmt.UUID]struct{}{},
	}

	if !fileutils.TargetExists(depotPath) {
		return result, nil
	}

	if fileutils.TargetExists(configFile) {
		b, err := fileutils.ReadFile(configFile)
		if err != nil {
			return nil, errs.Wrap(err, "failed to read depot file")
		}
		if err := json.Unmarshal(b, &result.config); err != nil {
			return nil, errs.Wrap(err, "failed to unmarshal depot file")
		}

		// Filter out artifacts that no longer exist (eg. user ran `state clean cache`)
		for id := range result.config.Deployments {
			if !fileutils.DirExists(result.Path(id)) {
				delete(result.config.Deployments, id)
			}
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

// DeployViaLink will take an artifact from the depot and link it to the target path.
func (d *depot) DeployViaLink(id strfmt.UUID, relativeSrc string) error {
	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	absoluteSrc := filepath.Join(d.Path(id), relativeSrc)
	if !fileutils.DirExists(absoluteSrc) {
		return errs.New("artifact src does not exist: %s", absoluteSrc)
	}

	// Copy or link the artifact files, depending on whether the artifact in question relies on file transformations
	if err := smartlink.LinkContents(absoluteSrc, d.targetPath); err != nil {
		return errs.Wrap(err, "failed to link artifact")
	}

	// Record deployment to config
	if _, ok := d.config.Deployments[id]; !ok {
		d.config.Deployments[id] = []deployment{}
	}
	d.config.Deployments[id] = append(d.config.Deployments[id], deployment{Type: deploymentTypeLink})

	return nil
}

// DeployViaCopy will take an artifact from the depot and copy it to the target path.
func (d *depot) DeployViaCopy(id strfmt.UUID, relativeSrc string) error {
	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	absoluteSrc := filepath.Join(d.Path(id), relativeSrc)
	if !fileutils.DirExists(absoluteSrc) {
		return errs.New("artifact src does not exist: %s", absoluteSrc)
	}

	// Copy or link the artifact files, depending on whether the artifact in question relies on file transformations
	if err := fileutils.CopyFiles(absoluteSrc, d.targetPath); err != nil {
		var errExist *fileutils.ErrAlreadyExist
		if errors.As(err, &errExist) {
			logging.Warning("Skipping files that already exist: " + errs.JoinMessage(errExist))
		} else {
			return errs.Wrap(err, "failed to copy artifact")
		}
	}

	// Record deployment to config
	if _, ok := d.config.Deployments[id]; !ok {
		d.config.Deployments[id] = []deployment{}
	}
	d.config.Deployments[id] = append(d.config.Deployments[id], deployment{Type: deploymentTypeCopy})

	return nil
}

func (d *depot) Undeploy(id strfmt.UUID, relativeSrc, path string) error {
	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	// Find record of our deployment
	if _, ok := d.config.Deployments[id]; !ok {
		return errs.New("deployment for %s not found in depot", id)
	}

	// Perform uninstall based on deployment type
	if err := smartlink.UnlinkContents(filepath.Join(d.Path(id), relativeSrc), path); err != nil {
		return errs.Wrap(err, "failed to unlink artifact")
	}

	// Write changes to config
	delete(d.config.Deployments, id)

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
	configFile := filepath.Join(d.targetPath, configDir, depotFile)
	b, err := json.Marshal(d.config)
	if err != nil {
		return errs.Wrap(err, "failed to marshal depot file")
	}
	if err := fileutils.WriteFile(configFile, b); err != nil {
		return errs.Wrap(err, "failed to write depot file")
	}
	return nil
}

func (d *depot) List() map[strfmt.UUID]struct{} {
	result := map[strfmt.UUID]struct{}{}
	for id, _ := range d.config.Deployments {
		result[id] = struct{}{}
	}

	return result
}