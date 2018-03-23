package artifact

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/fileutils"
)

// Meta is used to describe the contents of an artifact, this information can then be used to set up distributions
// An artifact can be a package, a language, a clib, etc. It is a component that makes up a distribution.
// It reflects the spec described here https://docs.google.com/document/d/1HprLsYXiKBeKfUvRrXpgyD_aodMnf6ZyuwgqpZu5ii4
type Meta struct {
	Name     string
	Type     string
	Version  string
	Relocate string
	Binaries []string
}

// Artifact describes the actual artifact as it is stored on the system
type Artifact struct {
	Meta *Meta
	Path string
	Hash string
}

// Get retrieves an artifact by the given hash
func Get(hash string) (*Artifact, *failures.Failure) {
	path := GetPath(hash)

	if !strings.HasSuffix(path, constants.ArtifactFile) {
		path = filepath.Join(path, constants.ArtifactFile)
	}

	if !fileutils.FileExists(path) {
		return nil, failures.FailNotFound.New("Artifact file does not exist at " + path)
	}

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	artf := Meta{}
	err = json.Unmarshal(body, &artf)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return &Artifact{&artf, path, hash}, nil
}

// GetPath retrieves the path for an artifact hash
func GetPath(hash string) string {
	datadir := config.GetDataDir()
	return filepath.Join(datadir, "artifacts", hash)
}

// Exists checks if an artifact exists by the given hash
func Exists(hash string) bool {
	path := GetPath(hash)

	if !strings.HasSuffix(path, constants.ArtifactFile) {
		path = filepath.Join(path, constants.ArtifactFile)
	}

	return fileutils.FileExists(path)
}
