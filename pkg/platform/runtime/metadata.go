package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	// FailMetaDataNotDetected indicates a failure due to the metafile not being detected.
	FailMetaDataNotDetected = failures.Type("runtime.metadata.notdetected", failures.FailIO, failures.FailNotFound)
)

// TargetedRelocation is a relocation instruction for files in a specific directory
type TargetedRelocation struct {
	// InDir is the directory in which files need to be relocated
	InDir string
	// SearchString to be replaced
	SearchString string
	// Replacement is the replacement string
	Replacement string
}

// MetaData is used to parse the metadata.json file
type MetaData struct {
	// Path is the directory containing the meta file
	Path string

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
	TargetedRelocations []TargetedRelocation
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
func InitMetaData(installDir string) (*MetaData, *failures.Failure) {
	var metaData *MetaData
	metaFile := filepath.Join(installDir, constants.RuntimeMetaFile)
	if fileutils.FileExists(metaFile) {
		contents, fail := fileutils.ReadFile(metaFile)
		if fail != nil {
			return nil, fail
		}

		metaData, fail = ParseMetaData(contents)
		if fail != nil {
			return nil, fail
		}
	} else {
		metaData = &MetaData{}
	}

	if metaData.Env == nil {
		metaData.Env = map[string]string{}
	}

	metaData.Path = installDir
	fail := metaData.Prepare()
	if fail != nil {
		return nil, fail
	}

	return metaData, nil
}

// ParseMetaData will parse the given bytes into the MetaData struct
func ParseMetaData(contents []byte) (*MetaData, *failures.Failure) {
	metaData := &MetaData{
		Env: make(map[string]string),
	}
	err := json.Unmarshal(contents, metaData)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	// The JSON decoder does not recognize 0 and 1 as bools, so we have to get crafty
	for k := range metaData.BinaryLocations {
		metaData.BinaryLocations[k].Relative = metaData.BinaryLocations[k].RelativeInt == 1
	}

	return metaData, nil
}

func (m *MetaData) hasBinaryFile(executable string) bool {
	for _, dir := range m.BinaryLocations {
		parent := ""
		if dir.Relative {
			parent = m.Path
		}
		bin := filepath.Join(parent, dir.Path, executable)
		if fileutils.FileExists(bin) {
			return true
		}
	}

	return false
}

func (m *MetaData) setPythonEnv() {
	if _, exists := m.Env["PYTHONPATH"]; !exists {
		m.Env["PYTHONPATH"] = "{{.ProjectDir}}"
	} else {
		logging.Debug("Not setting PYTHONPATH as the user already has it set")
	}

	if os.Getenv("PYTHONIOENCODING") == "" {
		m.Env["PYTHONIOENCODING"] = "utf-8"
	} else {
		logging.Debug("Not setting PYTHONIOENCODING as the user already has it set")
	}
}
