package ecosystem

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/unarchiver"

	"github.com/ActiveState/cli/pkg/buildplan"
)

type DotNet struct {
	runtimePath string
	nupkgDir    string
}

func (e *DotNet) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimePath = runtimePath
	e.nupkgDir = filepath.Join("usr", "nupkg")
	err := fileutils.MkdirUnlessExists(filepath.Join(e.runtimePath, e.nupkgDir))
	if err != nil {
		return errs.Wrap(err, "Unable to create nupkg directory")
	}
	return nil
}

func (e *DotNet) Namespaces() []string {
	return []string{"language/dotnet"}
}

// Note: it's okay to leave the last two values blank.
const nupkgMetadata = `{
"version": 2,
"contentHash": "",
"source": ""
}`

func (e *DotNet) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	installedFiles := []string{}

	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}
	for _, file := range files {
		if file.Name() == "runtime.json" {
			err = injectEnvVar(file.AbsolutePath(), "NUGET_PACKAGES", "${INSTALLDIR}/"+e.nupkgDir)
			if err != nil {
				return nil, errs.Wrap(err, "Unable to add NUGET_PACKAGES to runtime.json")
			}
			continue
		}
		if !strings.HasSuffix(file.Name(), ".nupkg") {
			continue
		}

		// The .NET runtime's package tree looks like this:
		// $NUGET_PACKAGES/
		//   - name1/
		//     - version1/
		//     - version2/
		//   - name2/
		//     - version1/
		//     - version2/
		// ...
		// Create the <name> folder so the nupkg can be unpacked into <name>/<version>.
		relativeNupkgDir := filepath.Join(e.nupkgDir, artifact.Name(), artifact.Version())
		absNupkgDir := filepath.Join(e.runtimePath, relativeNupkgDir)
		err = fileutils.MkdirUnlessExists(filepath.Dir(absNupkgDir))
		if err != nil {
			return nil, errs.Wrap(err, "Unable to create nupkg dir")
		}

		// Delete any previously extracted package.
		if fileutils.DirExists(absNupkgDir) {
			err = os.RemoveAll(absNupkgDir)
			if err != nil {
				return nil, errs.Wrap(err, "Unable to remove previously unpacked nupkg")
			}
		}

		// Unpack the nupkg into a <version> folder inside the <name> folder.
		ua := unarchiver.NewZip()
		f, err := ua.PrepareUnpacking(file.AbsolutePath(), absNupkgDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to prepare for unpacking downloaded nupkg")
		}
		err = ua.Unarchive(f, absNupkgDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to unpack downloaded nupkg")
		}

		// Packages need a .nupkg.metadata file too.
		err = fileutils.WriteFile(filepath.Join(absNupkgDir, ".nupkg.metadata"), []byte(nupkgMetadata))
		if err != nil {
			return nil, errs.Wrap(err, "Unable to write empty metadata")
		}

		installedFiles = append(installedFiles, relativeNupkgDir)
	}

	return installedFiles, nil
}

func (e *DotNet) Remove(name, version string, installedFiles []string) (rerr error) {
	for _, dir := range installedFiles {
		if !fileutils.DirExists(dir) {
			continue
		}
		err := os.RemoveAll(dir)
		if err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "Unable to remove directory for '%s': %s", name, dir))
		}
	}
	return rerr
}

func (e *DotNet) Apply() error {
	return nil
}
