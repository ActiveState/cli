package runtime

import (
	"encoding/json"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

var (
	// FailMetaDataNotFound indicates a failure due to the metafile not being found. Kinda speaks for itself don't it? Silly golint.
	FailMetaDataNotFound = failures.Type("runtime.metadata.notfound", failures.FailIO, failures.FailNotFound)
)

// MetaData is used to parse the metadata.json file
type MetaData struct {

	// AffectedEnv is an environment variable that we should ensure is not set, as it might conflict with the artifact
	AffectedEnv string `json:"affected_env"`

	// BinaryLocations are locations that we should add to the PATH
	BinaryLocations []MetaDataBinary `json:"binaries_in"`

	// RelocationDir is the string that we should replace with the actual install dir of the artifact
	RelocationDir string `json:"relocation_dir"`
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
	metaFile := filepath.Join(installDir, constants.RuntimeMetaFile)
	if !fileutils.FileExists(metaFile) {
		return nil, FailMetaDataNotFound.New("installer_err_runtime_missing_meta_file", installDir, constants.RuntimeMetaFile)
	}

	contents, fail := fileutils.ReadFile(metaFile)
	if fail != nil {
		return nil, fail
	}

	return ParseMetaData(contents)
}

// ParseMetaData will parse the given bytes into the MetaData struct
func ParseMetaData(contents []byte) (*MetaData, *failures.Failure) {
	metaData := &MetaData{}
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
