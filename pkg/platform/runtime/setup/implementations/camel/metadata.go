package camel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type ErrMetaData struct{ *errs.WrapperError }

// TargetedRelocation is a relocation instruction for files in a specific directory
type TargetedRelocation struct {
	// InDir is the directory in which files need to be relocated
	InDir string `json:"dir"`
	// SearchString to be replaced
	SearchString string `json:"search"`
	// Replacement is the replacement string
	Replacement string `json:"replace"`
}

// MetaData is used to parse the metadata.json file
type MetaData struct {
	// InstallDir is the root directory of the artifact files that we need to copy on the user's machine
	InstallDir string

	// AffectedEnv is an environment variable that we should ensure is not set, as it might conflict with the artifact
	AffectedEnv string `json:"affected_env"`

	// Env is a key value map containing all the env vars, values can contain the RelocationDir value (which will be replaced)
	Env map[string]string `json:"env"`

	// BinaryLocations are locations that we should add to the PATH
	BinaryLocations []MetaDataBinary `json:"binaries_in"`

	// RelocationDir is the string that we should replace with the actual install dir of the artifact
	RelocationDir string `json:"relocation_dir"`

	// LibLocation is the place in which .so and .dll files are stored (which binary files will need relocated)
	RelocationTargetBinaries string `json:"relocation_target_binaries"`

	// TargetedRelocations are relocations that only target specific parts of the installation
	TargetedRelocations []TargetedRelocation `json:"custom_relocations"`
}

// MetaDataBinary is used to represent a binary path contained within the metadata.json file
type MetaDataBinary struct {
	Path     string `json:"path"`
	Relative bool

	// RelativeInt is used to unmarshal the 'relative' boolean, which is given as a 0 or a 1, which Go's
	// json package doesn't recognize as bools.
	// Don't use this field, use Relative instead.
	RelativeInt int `json:"relative"`
}

// InitMetaData will create an instance of MetaData based on the metadata.json file found under the given artifact install dir
func InitMetaData(rootDir string) (*MetaData, error) {
	var metaData *MetaData
	metaFile := filepath.Join(rootDir, "support", constants.RuntimeMetaFile)
	if fileutils.FileExists(metaFile) {
		contents, err := fileutils.ReadFile(metaFile)
		if err != nil {
			return nil, err
		}

		metaData, err = ParseMetaData(contents)
		if err != nil {
			return nil, err
		}
	} else {
		metaData = &MetaData{}
	}

	if metaData.Env == nil {
		metaData.Env = map[string]string{}
	}

	var relInstallDir string
	installDirs := strings.Split(constants.RuntimeInstallDirs, ",")
	for _, dir := range installDirs {
		if fileutils.DirExists(filepath.Join(rootDir, dir)) {
			relInstallDir = dir
		}
	}

	if relInstallDir == "" {
		logging.Debug("Did not find an installation directory relative to metadata file.")
	}

	metaData.InstallDir = relInstallDir
	err := metaData.Prepare(filepath.Join(rootDir, relInstallDir))
	if err != nil {
		return nil, err
	}

	return metaData, nil
}

// ParseMetaData will parse the given bytes into the MetaData struct
func ParseMetaData(contents []byte) (*MetaData, error) {
	metaData := &MetaData{
		Env: make(map[string]string),
	}
	err := json.Unmarshal(contents, metaData)
	if err != nil {
		return nil, &ErrMetaData{errs.Wrap(err, "Unmarshal failed")}
	}

	// The JSON decoder does not recognize 0 and 1 as bools, so we have to get crafty
	for k := range metaData.BinaryLocations {
		metaData.BinaryLocations[k].Relative = metaData.BinaryLocations[k].RelativeInt == 1
	}

	return metaData, nil
}

func (m *MetaData) hasBinaryFile(root string, executable string) bool {
	for _, dir := range m.BinaryLocations {
		parent := ""
		if dir.Relative {
			parent = root
		}
		bin := filepath.Join(parent, dir.Path, executable)
		if fileutils.FileExists(bin) {
			return true
		}
	}

	return false
}

func (m *MetaData) setPythonEnv() {
	// This is broken for two reasons:
	// 1. Checking in the OS environment will only happen on installation, but at a later point, the OS environment might have changed, and we will overwrite the user's choice here
	// 2. python code does not need to depend on PYTHONIOENCODING as pointed out here: https://stackoverflow.com/a/9942822
	// Follow up story is here: https://www.pivotaltracker.com/story/show/177407383
	if os.Getenv("PYTHONIOENCODING") == "" {
		m.Env["PYTHONIOENCODING"] = "utf-8"
	} else {
		logging.Debug("Not setting PYTHONIOENCODING as the user already has it set")
	}
}
