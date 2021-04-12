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
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
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

	// First: We need to identify the values for <archiveName> and <relInstallDir>

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

	// parse the legacy metadata
	md, err := InitMetaData(tmpBaseDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not load meta data definitions for camel artifact.")
	}

	// convert file relocation commands into an envdef.FileTransform slice
	transforms, err := convertToFileTransforms(tmpBaseDir, md.InstallDir, md)
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine file transformations")
	}

	// convert environment variables into an envdef.EnvironmentVariable slice
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
			Values:  []string{},
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

	return res
}

func paddingForBinaryFile(isBinary bool) *string {
	if !isBinary {
		return nil
	}
	pad := "\000"
	return &pad
}

func convertToFileTransforms(tmpBaseDir string, relInstDir string, metadata *MetaData) ([]envdef.FileTransform, error) {
	var res []envdef.FileTransform
	instDir := filepath.Join(tmpBaseDir, relInstDir)
	for _, tr := range metadata.TargetedRelocations {
		// walk through files in tr.InDir and find files that need replacements. For those we create a FileTransform element
		trans, err := fileTransformsInDir(instDir, filepath.Join(instDir, tr.InDir), tr.SearchString, tr.Replacement, func(_ string, _ bool) bool { return true })
		if err != nil {
			return res, errs.Wrap(err, "Failed convert targeted relocations")
		}
		res = append(res, trans...)
	}

	// metadata.RelocationDir is the string to search for and replace with ${INSTALLDIR}
	if metadata.RelocationDir == "" {
		return res, nil
	}
	binariesSeparate := runtime.GOOS == "linux" && metadata.RelocationTargetBinaries != ""

	relocFilePath := filepath.Join(tmpBaseDir, "support", "reloc.txt")
	relocMap := map[string]bool{}
	if fileutils.FileExists(relocFilePath) {
		relocMap = loadRelocationFile(relocFilePath)
	}

	trans, err := fileTransformsInDir(instDir, instDir, metadata.RelocationDir, "${INSTALLDIR}", func(path string, isBinary bool) bool {
		return relocMap[path] || !binariesSeparate || !isBinary
	})
	if err != nil {
		return res, errs.Wrap(err, "Could not determine transformations in installation directory")
	}
	res = append(res, trans...)

	if binariesSeparate {
		trans, err := fileTransformsInDir(instDir, instDir, metadata.RelocationDir, "${INSTALLDIR}", func(_ string, isBinary bool) bool {
			return isBinary
		})
		if err != nil {
			return res, errs.Wrap(err, "Could not determine separate binary transformations in installation directory")
		}
		res = append(res, trans...)
	}
	return res, nil
}

// fileTransformsInDir walks through all the files in searchDir and creates a FileTransform item for files that contain searchString and pass the filter function
func fileTransformsInDir(instDir string, searchDir string, searchString string, replacement string, filter func(string, bool) bool) ([]envdef.FileTransform, error) {
	var res []envdef.FileTransform

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return errs.Wrap(err, "Could not read file path %s", path)
		}

		// relativePath is the path relative to the installation directory
		relativePath := strings.TrimPrefix(path, instDir)
		isBinary := fileutils.IsBinary(b)
		if !filter(relativePath, isBinary) {
			return nil
		}
		if bytes.Contains(b, []byte(searchString)) {
			res = append(res, envdef.FileTransform{
				In:      []string{relativePath},
				Pattern: searchString,
				With:    replacement,
				PadWith: paddingForBinaryFile(isBinary),
			})
		}

		return nil
	})
	return res, err
}

func (as *ArtifactSetup) Unarchiver() unarchiver.Unarchiver {
	if runtime.GOOS == "windows" {
		return unarchiver.NewZip()
	}
	return unarchiver.NewTarGz()
}
