package ecosystem

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/smartlink"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/buildplan"
)

type Golang struct {
	runtimeDir          string
	proxyDir            string
	addedModuleVersions map[string][]string
}

func (e *Golang) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimeDir = runtimePath
	e.proxyDir = filepath.Join("usr", "goproxy")
	err := fileutils.MkdirUnlessExists(filepath.Join(e.runtimeDir, e.proxyDir))
	if err != nil {
		return errs.Wrap(err, "Unable to create Go proxy directory")
	}
	e.addedModuleVersions = make(map[string][]string)
	return nil
}

func (e *Golang) Namespaces() []string {
	return []string{"language/golang"}
}

// Unpack the module into the proxy directory.
// We also inject the GOPROXY environment variable into runtime.json to force offline use.
// We also inject GOMODCACHE to avoid polluting the default user cache.
func (e *Golang) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	installedFiles := []string{}

	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}

	for _, file := range files {
		if file.Name() == "runtime.json" {
			err = injectEnvVar(file.AbsolutePath(), "GOPROXY", "file://${INSTALLDIR}/usr/goproxy")
			if err != nil {
				return nil, errs.Wrap(err, "Unable to add GOPROXY to runtime.json")
			}
			err = injectEnvVar(file.AbsolutePath(), "GOMODCACHE", "${INSTALLDIR}/usr/goproxy/cache")
			if err != nil {
				return nil, errs.Wrap(err, "Unable to add GOMODCACHE to runtime.json")
			}
			continue
		}
		if !strings.HasSuffix(file.Name(), ".zip") {
			continue
		}

		// The structure of a Go proxy is:
		// proxydir/
		// - example.com/
		//   - mymodule/
		//     - @v/
		//       - list
		//	      - v1.0.0.mod
		//	      - v1.0.0.zip

		relativeProxied := filepath.Join(e.proxyDir, artifact.Name())
		absProxied := filepath.Join(e.runtimeDir, relativeProxied)

		// Create the @v directory if it doesn't already exist.
		vDir := filepath.Join(absProxied, "@v")
		err = fileutils.MkdirUnlessExists(vDir)
		if err != nil {
			return nil, errs.Wrap(err, "Could not create proxy module @v directory")
		}

		// Link/copy the zip file into the @v directory.
		err = smartlink.Link(file.AbsolutePath(), filepath.Join(vDir, file.Name()))

		// Extract the go.mod from the zip and copy it into the @v directory with a versioned name.
		ua := unarchiver.NewZip()
		unpackDir := fileutils.TempFilePath("", "")
		f, size, err := ua.PrepareUnpacking(file.AbsolutePath(), unpackDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to prepare for unpacking downloaded module")
		}
		err = ua.Unarchive(f, size, unpackDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to unpack downloaded module")
		}
		err = fileutils.CopyFile(filepath.Join(unpackDir, artifact.NameAndVersion(), "go.mod"), filepath.Join(vDir, artifact.Version()+".mod"))
		if err != nil {
			return nil, errs.Wrap(err, "Unable to copy go.mod from unpacked module")
		}
		err = os.RemoveAll(unpackDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to remove unpacked module")
		}

		installedFiles = append(installedFiles, relativeProxied)
	}

	e.addedModuleVersions[artifact.Name()] = append(e.addedModuleVersions[artifact.Name()], artifact.Version())

	return installedFiles, nil
}

func (e *Golang) Remove(artifact *buildplan.Artifact) error {
	return nil // TODO: CP-956
}

// Create/update each added module's version list file.
func (e *Golang) Apply() error {
	for name, versions := range e.addedModuleVersions {
		listFile := filepath.Join(e.runtimeDir, e.proxyDir, name, "@v", "list")
		if fileutils.FileExists(listFile) {
			// Add known versions to the new versions so we can write a comprehensive list.
			contents, err := fileutils.ReadFile(listFile)
			if err != nil {
				return errs.Wrap(err, "Unable to read %s", listFile)
			}
			for _, version := range strings.Split(string(contents), "\n") {
				versions = append(versions, version)
			}
		}

		// Sort versions in descending order by semver.
		sort.SliceStable(versions, func(i, j int) bool {
			return semver.Compare(versions[i], versions[j]) < 0
		})

		// Write all known versions.
		err := fileutils.WriteFile(listFile, []byte(strings.Join(versions, "\n")))
		if err != nil {
			return errs.Wrap(err, "Unable to write %s", listFile)
		}
	}
	return nil
}
