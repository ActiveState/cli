package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	// FailMetaDataNotDetected indicates a failure due to the metafile not being detected.
	FailMetaDataNotDetected = failures.Type("runtime.metadata.notdetected", failures.FailIO, failures.FailNotFound)
)

const (
	pythonEncodingEnvVar = "PYTHONIOENCODING"
	pythonEncodingUTF8   = "utf-8"
)

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
	fail := metaData.MakeBackwardsCompatible()
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

// MakeBackwardsCompatible will assume the LibLocation in cases where the metadata
// doesn't contain it and we know what it should be
func (m *MetaData) MakeBackwardsCompatible() *failures.Failure {
	// BinaryLocations
	if m.BinaryLocations == nil || len(m.BinaryLocations) == 0 {
		m.BinaryLocations = []MetaDataBinary{
			MetaDataBinary{
				Path:     "bin",
				Relative: true,
			},
		}
	}

	// Python
	if m.hasBinaryFile(constants.ActivePython3Executable) || m.hasBinaryFile(constants.ActivePython2Executable) {
		logging.Debug("Detected Python artifact, ensuring backwards compatibility")

		// RelocationTargetBinaries
		if m.RelocationTargetBinaries == "" {
			if runtime.GOOS == "windows" {
				m.RelocationTargetBinaries = "DLLs"
			} else {
				m.RelocationTargetBinaries = "lib"
			}
		}
		// RelocationDir
		if m.RelocationDir == "" {
			var fail *failures.Failure
			if m.RelocationDir, fail = m.pythonRelocationDir(); fail != nil {
				return fail
			}
		}
		// Env
		if _, exists := m.Env["PYTHONPATH"]; !exists {
			m.Env["PYTHONPATH"] = "{{.ProjectDir}}"
		}
		if os.Getenv(pythonEncodingEnvVar) == "" {
			m.Env[pythonEncodingEnvVar] = pythonEncodingUTF8
		}

		//Perl
	} else if m.hasBinaryFile(constants.ActivePerlExecutable) {
		logging.Debug("Detected Perl artifact, ensuring backwards compatibility")

		// RelocationDir
		if m.RelocationDir == "" {
			var fail *failures.Failure
			if m.RelocationDir, fail = m.perlRelocationDir(); fail != nil {
				return fail
			}
		}
		// AffectedEnv
		if m.AffectedEnv == "" {
			m.AffectedEnv = "PERL5LIB"
		}
	} else {
		logging.Debug("No language detected for %s", m.Path)
	}

	if m.RelocationDir == "" {
		return FailMetaDataNotDetected.New("installer_err_runtime_missing_meta")
	}

	return nil
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
