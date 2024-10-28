package camel

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/runtime/internal/envdef"
	"github.com/thoas/go-funk"
)

func NewEnvironmentDefinitions(rootDir string) (*envdef.EnvironmentDefinition, error) {
	dirEntries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not read directory")
	}
	if len(dirEntries) != 1 {
		return nil, errs.New("Camel artifacts are expected to have a single directory at its root")
	}

	baseDir := dirEntries[0].Name()
	absoluteBaseDir := filepath.Join(rootDir, baseDir)

	meta, err := newMetaData(absoluteBaseDir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine metadata")
	}

	fileTransforms, err := convertToFileTransforms(absoluteBaseDir, meta)
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine file transforms")
	}

	return &envdef.EnvironmentDefinition{
		Env:        convertToEnvVars(meta),
		Transforms: fileTransforms,
		InstallDir: filepath.Join(baseDir, meta.InstallDir),
	}, nil
}

func convertToEnvVars(metadata *metaData) []envdef.EnvironmentVariable {
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
	for k, v := range metadata.PathListEnv {
		res = append(res, envdef.EnvironmentVariable{
			Name:      k,
			Values:    []string{v},
			Join:      envdef.Prepend,
			Separator: string(os.PathListSeparator),
			Inherit:   true,
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

func convertToFileTransforms(tmpBaseDir string, metadata *metaData) ([]envdef.FileTransform, error) {
	var res []envdef.FileTransform
	instDir := filepath.Join(tmpBaseDir, metadata.InstallDir)
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

		// skip symlinks
		if (info.Mode() & fs.ModeSymlink) == fs.ModeSymlink {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		b, err := os.ReadFile(path)
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
