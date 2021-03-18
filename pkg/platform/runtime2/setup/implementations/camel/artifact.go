package camel

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime2/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime2/store"
	"github.com/thoas/go-funk"
)

type ErrNotExecutable struct{ *locale.LocalizedError }

type ErrNoPrefixes struct{ *locale.LocalizedError }

type ArtifactSetup struct {
	artifactID artifact.ArtifactID
	store      *store.Store
}

func NewArtifactSetup(artifactID artifact.ArtifactID, store *store.Store) *ArtifactSetup {
	return &ArtifactSetup{artifactID, store}
}

func (as *ArtifactSetup) EnvDef(tmpDir string) (*envdef.EnvironmentDefinition, error) {
	// camel archives are structured like this
	// <archiveName>/
	//    <relInstallDir>/
	//       artifact contents ...
	//    metadata.json
	//
	// We need to identify the values for <archiveName> and <relInstallDir>

	var archiveName string
	fs, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not read temporary installation directory %s", tmpDir)
	}
	for _, f := range fs {
		if f.IsDir() {
			archiveName = f.Name()
		}
	}
	if archiveName == "" {
		return nil, errs.New("Expected sub-directory in extracted artifact tarball.")
	}

	tmpBaseDir := filepath.Join(tmpDir, archiveName)

	md, err := InitMetaData(tmpBaseDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not load meta data definitions for camel artifact.")
	}

	transforms, err := convertToFileTransforms(tmpBaseDir, md.InstallDir, md)
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine file transformations")
	}

	vars := convertToEnvVars(md)

	ed := &envdef.EnvironmentDefinition{
		InstallDir: filepath.Join(archiveName, md.InstallDir),
		Transforms: transforms,
		Env:        vars,
	}

	return ed, nil
}

func convertToEnvVars(metadata *MetaData) []envdef.EnvironmentVariable {
	var res []envdef.EnvironmentVariable
	if metadata.AffectedEnv != "" {
		res = append(res, envdef.EnvironmentVariable{
			Name:    metadata.AffectedEnv,
			Inherit: false,
			Join:    envdef.Disallowed,
		})
	}
	for k, v := range metadata.Env {
		res = append(res, envdef.EnvironmentVariable{
			Name:    k,
			Values:  []string{v},
			Inherit: false,
		})
	}
	var binPaths []string

	// set up PATH according to binary locations
	for _, v := range metadata.BinaryLocations {
		path := v.Path
		if v.Relative {
			path = filepath.Join("${INSTALLDIR}", path)
		}
		binPaths = append(binPaths, path)
	}

	// Add DLL dir to PATH on Windows
	if runtime.GOOS == "windows" && metadata.RelocationTargetBinaries != "" {
		binPaths = append(binPaths, filepath.Join("${INSTALLDIR}", metadata.RelocationTargetBinaries))

	}

	res = append(res, envdef.EnvironmentVariable{
		Name:      "PATH",
		Values:    funk.ReverseStrings(binPaths),
		Inherit:   true,
		Join:      envdef.Prepend,
		Separator: string(os.PathListSeparator),
	})

	res = append(res, metadata.ExtraVariables...)

	return res
}

func convertToFileTransforms(tmpBaseDir string, relInstDir string, metadata *MetaData) ([]envdef.FileTransform, error) {
	var res []envdef.FileTransform
	instDir := filepath.Join(tmpBaseDir, relInstDir)
	for _, tr := range metadata.TargetedRelocations {
		err := filepath.Walk(tr.InDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errs.Wrap(err, "Error walking tree for targeted relocations")
			}

			if info.IsDir() {
				return nil
			}
			trimmed := strings.TrimPrefix(path, instDir)

			b, err := ioutil.ReadFile(path)
			if err != nil {
				return errs.Wrap(err, "Could not read file path %s", path)
			}
			var padWith *string
			if fileutils.IsBinary(b) {
				pad := "\000"
				padWith = &pad
			}
			if bytes.Contains(b, []byte(tr.SearchString)) {
				res = append(res, envdef.FileTransform{
					In:      []string{trimmed},
					Pattern: tr.SearchString,
					With:    tr.Replacement,
					PadWith: padWith,
				})
			}

			return nil
		})
		if err != nil {
			return res, errs.Wrap(err, "Failed convert targeted relocations")
		}
	}

	if metadata.RelocationDir == "" {
		return res, nil
	}

	relocFilePath := filepath.Join(tmpBaseDir, "support", "reloc.txt")
	relocMap := map[string]bool{}
	if fileutils.FileExists(relocFilePath) {
		relocMap = loadRelocationFile(relocFilePath)
	}

	err := filepath.Walk(instDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		trimmed := strings.TrimPrefix(path, instDir)

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return errs.Wrap(err, "Could not read file path %s", path)
		}
		var padWith *string
		var with string = "${INSTALLDIR}"
		if fileutils.IsBinary(b) {
			pad := "\000"
			padWith = &pad
			with = filepath.Join("${INSTALLDIR}", metadata.RelocationTargetBinaries)
		}
		if relocMap[trimmed] || bytes.Contains(b, []byte(metadata.RelocationDir)) {
			res = append(res, envdef.FileTransform{
				In:      []string{trimmed},
				Pattern: metadata.RelocationDir,
				With:    with,
				PadWith: padWith,
			})
		}

		return nil
	})
	if err != nil {
		return nil, errs.Wrap(err, "Failed to inspect temporary installation directory %s for relocatable files", instDir)
	}
	return res, nil
}

func (as *ArtifactSetup) Unarchiver() unarchiver.Unarchiver {
	if runtime.GOOS == "windows" {
		return unarchiver.NewZip()
	}
	return unarchiver.NewTarGz()
}

func (as *ArtifactSetup) InstallerExtension() string {
	if runtime.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
