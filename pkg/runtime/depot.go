package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/smartlink"
)

const (
	depotFile = "depot.json"
)

type depotConfig struct {
	Deployments map[strfmt.UUID][]deployment `json:"deployments"`
}

type deployment struct {
	Type        deploymentType `json:"type"`
	Path        string         `json:"path"`
	Files       []string       `json:"files"`
	RelativeSrc string         `json:"relativeSrc"`
}

type deploymentType string

const (
	deploymentTypeLink deploymentType = "link"
	deploymentTypeCopy                = "copy"
)

type ErrVolumeMismatch struct {
	DepotVolume string
	PathVolume  string
}

func (e ErrVolumeMismatch) Error() string {
	return fmt.Sprintf("volume mismatch: path volume is '%s', but depot volume is '%s'", e.PathVolume, e.DepotVolume)
}

type depot struct {
	config    depotConfig
	depotPath string
	artifacts map[strfmt.UUID]struct{}
	fsMutex   *sync.Mutex
}

func newDepot(runtimePath string) (*depot, error) {
	depotPath := filepath.Join(storage.CachePath(), depotName)

	// Windows does not support hard-linking across drives, so determine if the runtime path is on a
	// separate drive than the default depot path. If so, use a drive-specific depot path.
	if runtime.GOOS == "windows" {
		runtimeVolume := filepath.VolumeName(runtimePath)
		storageVolume := filepath.VolumeName(storage.CachePath())
		if runtimeVolume != storageVolume {
			depotPath = filepath.Join(runtimeVolume+"\\", "activestate", depotName)
		}
	}

	result := &depot{
		config: depotConfig{
			Deployments: map[strfmt.UUID][]deployment{},
		},
		depotPath: depotPath,
		artifacts: map[strfmt.UUID]struct{}{},
		fsMutex:   &sync.Mutex{},
	}

	if !fileutils.TargetExists(depotPath) {
		return result, nil
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

		// Filter out deployments that no longer exist (eg. user ran `state clean cache`)
		for id, deployments := range result.config.Deployments {
			if !fileutils.DirExists(result.Path(id)) {
				delete(result.config.Deployments, id)
				continue
			}
			result.config.Deployments[id] = sliceutils.Filter(deployments, func(d deployment) bool {
				return someFilesExist(d.Files, d.Path)
			})
		}
	}

	files, err := os.ReadDir(depotPath)
	if err != nil {
		return nil, errs.Wrap(err, "failed to read depot path")
	}

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
func (d *depot) DeployViaLink(id strfmt.UUID, relativeSrc, absoluteDest string) error {
	d.fsMutex.Lock()
	defer d.fsMutex.Unlock()

	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	if err := d.validateVolume(absoluteDest); err != nil {
		return errs.Wrap(err, "volume validation failed")
	}

	// Collect artifact meta info
	var err error
	absoluteDest, err = fileutils.ResolvePath(absoluteDest)
	if err != nil {
		return errs.Wrap(err, "failed to resolve path")
	}

	if err := fileutils.MkdirUnlessExists(absoluteDest); err != nil {
		return errs.Wrap(err, "failed to create path")
	}

	absoluteSrc := filepath.Join(d.Path(id), relativeSrc)
	if !fileutils.DirExists(absoluteSrc) {
		return errs.New("artifact src does not exist: %s", absoluteSrc)
	}

	// Copy or link the artifact files, depending on whether the artifact in question relies on file transformations
	if err := smartlink.LinkContents(absoluteSrc, absoluteDest); err != nil {
		return errs.Wrap(err, "failed to link artifact")
	}

	files, err := fileutils.ListDir(absoluteSrc, false)
	if err != nil {
		return errs.Wrap(err, "failed to list files")
	}

	// Record deployment to config
	if _, ok := d.config.Deployments[id]; !ok {
		d.config.Deployments[id] = []deployment{}
	}
	d.config.Deployments[id] = append(d.config.Deployments[id], deployment{
		Type:        deploymentTypeLink,
		Path:        absoluteDest,
		Files:       files.RelativePaths(),
		RelativeSrc: relativeSrc,
	})

	return nil
}

// DeployViaCopy will take an artifact from the depot and copy it to the target path.
func (d *depot) DeployViaCopy(id strfmt.UUID, relativeSrc, absoluteDest string) error {
	d.fsMutex.Lock()
	defer d.fsMutex.Unlock()

	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	var err error
	absoluteDest, err = fileutils.ResolvePath(absoluteDest)
	if err != nil {
		return errs.Wrap(err, "failed to resolve path")
	}

	if err := d.validateVolume(absoluteDest); err != nil {
		return errs.Wrap(err, "volume validation failed")
	}

	if err := fileutils.MkdirUnlessExists(absoluteDest); err != nil {
		return errs.Wrap(err, "failed to create path")
	}

	absoluteSrc := filepath.Join(d.Path(id), relativeSrc)
	if !fileutils.DirExists(absoluteSrc) {
		return errs.New("artifact src does not exist: %s", absoluteSrc)
	}

	// Copy or link the artifact files, depending on whether the artifact in question relies on file transformations
	if err := fileutils.CopyFiles(absoluteSrc, absoluteDest); err != nil {
		var errExist *fileutils.ErrAlreadyExist
		if errors.As(err, &errExist) {
			logging.Warning("Skipping files that already exist: " + errs.JoinMessage(errExist))
		} else {
			return errs.Wrap(err, "failed to copy artifact")
		}
	}

	files, err := fileutils.ListDir(absoluteSrc, false)
	if err != nil {
		return errs.Wrap(err, "failed to list files")
	}

	// Record deployment to config
	if _, ok := d.config.Deployments[id]; !ok {
		d.config.Deployments[id] = []deployment{}
	}
	d.config.Deployments[id] = append(d.config.Deployments[id], deployment{
		Type:        deploymentTypeCopy,
		Path:        absoluteDest,
		Files:       files.RelativePaths(),
		RelativeSrc: relativeSrc,
	})

	return nil
}

func (d *depot) Undeploy(id strfmt.UUID, relativeSrc, path string) error {
	d.fsMutex.Lock()
	defer d.fsMutex.Unlock()

	if !d.Exists(id) {
		return errs.New("artifact not found in depot")
	}

	var err error
	path, err = fileutils.ResolvePath(path)
	if err != nil {
		return errs.Wrap(err, "failed to resolve path")
	}

	// Find record of our deployment
	deployments, ok := d.config.Deployments[id]
	if !ok {
		return errs.New("deployment for %s not found in depot", id)
	}
	deployments = sliceutils.Filter(deployments, func(d deployment) bool { return d.Path == path })
	if len(deployments) != 1 {
		return errs.New("no deployment found for %s in depot", path)
	}
	deploy := deployments[0]

	// Perform uninstall based on deployment type
	if err := smartlink.UnlinkContents(filepath.Join(d.Path(id), relativeSrc), path); err != nil {
		return errs.Wrap(err, "failed to unlink artifact")
	}

	// Re-link or re-copy any files provided by other artifacts.
	redeploys, err := d.getSharedFilesToRedeploy(id, deploy, path)
	if err != nil {
		return errs.Wrap(err, "failed to get shared files")
	}
	for sharedFile, relinkSrc := range redeploys {
		switch deploy.Type {
		case deploymentTypeLink:
			if err := smartlink.Link(relinkSrc, sharedFile); err != nil {
				return errs.Wrap(err, "failed to relink file")
			}
		case deploymentTypeCopy:
			if err := fileutils.CopyFile(relinkSrc, sharedFile); err != nil {
				return errs.Wrap(err, "failed to re-copy file")
			}
		}
	}

	// Write changes to config
	d.config.Deployments[id] = sliceutils.Filter(d.config.Deployments[id], func(d deployment) bool { return d.Path != path })

	return nil
}

func (d *depot) validateVolume(absoluteDest string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	depotVolume := filepath.VolumeName(d.depotPath)
	pathVolume := filepath.VolumeName(absoluteDest)
	if pathVolume != depotVolume {
		return &ErrVolumeMismatch{depotVolume, pathVolume}
	}

	return nil
}

// getSharedFilesToRedeploy returns a map of deployed files to re-link to (or re-copy from) another
// artifact that provides those files. The key is the deployed file path and the value is the
// source path from another artifact.
func (d *depot) getSharedFilesToRedeploy(id strfmt.UUID, deploy deployment, path string) (map[string]string, error) {
	// Map of deployed paths to other sources that provides those paths.
	redeploy := make(map[string]string, 0)

	// For each file deployed by this artifact, find another artifact (if any) that deploys its own copy.
	for _, relativeDeployedFile := range deploy.Files {
		deployedFile := filepath.Join(path, relativeDeployedFile)
		for artifactId, artifactDeployments := range d.config.Deployments {
			if artifactId == id {
				continue
			}

			findArtifact := func() bool {
				for _, deployment := range artifactDeployments {
					for _, fileToDeploy := range deployment.Files {
						if relativeDeployedFile != fileToDeploy {
							continue
						}
						// We'll want to redeploy this from other artifact's copy after undeploying the currently deployed version.
						newSrc := filepath.Join(d.Path(artifactId), deployment.RelativeSrc, relativeDeployedFile)
						logging.Debug("More than one artifact provides '%s'", relativeDeployedFile)
						logging.Debug("Will redeploy '%s' to '%s'", newSrc, deployedFile)
						redeploy[deployedFile] = newSrc
						return true
					}
				}
				return false
			}

			if findArtifact() {
				break // ignore all other copies once one is found
			}
		}
	}

	return redeploy, nil
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

// someFilesExist will check up to 10 files from the given filepaths, if any of them exist it returns true.
// This is a temporary workaround for https://activestatef.atlassian.net/browse/DX-2913
// As of right now we cannot assert which artifact owns a given file, and so simply asserting if any one given file exists
// is inssuficient as an assertion.
func someFilesExist(filePaths []string, basePath string) bool {
	for x, filePath := range filePaths {
		if x == 10 {
			break
		}
		if fileutils.TargetExists(filepath.Join(basePath, filePath)) {
			return true
		}
	}
	return false
}
