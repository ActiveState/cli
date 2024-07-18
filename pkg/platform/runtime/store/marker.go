package store

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/go-openapi/strfmt"
)

type Marker struct {
	CommitID  string `json:"commitID"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

func (s *Store) markerFile() string {
	return filepath.Join(s.storagePath, constants.RuntimeInstallationCompleteMarker)
}

func (s *Store) HasMarker() bool {
	return fileutils.FileExists(s.markerFile())
}

// MarkerIsValid checks if stored runtime is complete and can be loaded
func (s *Store) MarkerIsValid(commitID strfmt.UUID) bool {
	marker, err := s.parseMarker()
	if err != nil {
		logging.Debug("Unable to parse marker file %s: %v", marker, err)
		return false
	}

	if marker.CommitID != commitID.String() {
		logging.Debug("Could not match commitID in %s, expected: %s, got: %s", marker, commitID.String(), marker.CommitID)
		return false
	}

	if marker.Version != constants.Version {
		logging.Debug("Could not match State Tool version in %s, expected: %s, got: %s", marker, constants.Version, marker.Version)
		return false
	}

	return true
}

// VersionMarkerIsValid checks if stored runtime was installed with the current state tool version
func (s *Store) VersionMarkerIsValid() bool {
	marker, err := s.parseMarker()
	if err != nil {
		logging.Debug("Unable to parse marker file %s: %v", marker, err)
		return false
	}

	if marker.Version != constants.Version {
		logging.Debug("Could not match State Tool version in %s, expected: %s, got: %s", marker, constants.Version, marker.Version)
		return false
	}

	return true
}

func (s *Store) parseMarker() (*Marker, error) {
	if !s.HasMarker() {
		return nil, errs.New(`Marker file "%s" does not exist`, s.markerFile())
	}

	contents, err := fileutils.ReadFile(s.markerFile())
	if err != nil {
		return nil, errs.Wrap(err, "Could not read marker file %s", s.markerFile())
	}

	if !json.Valid(contents) {
		return s.updateMarker(contents)
	}

	marker := &Marker{}
	err = json.Unmarshal(contents, marker)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmasrshal marker file")
	}

	return marker, nil
}

// updateMarker updates old marker files to the new format and
// returns the stored marker data
func (s *Store) updateMarker(contents []byte) (*Marker, error) {
	lines := strings.Split(string(contents), "\n")
	if len(lines) == 0 {
		// No marker data, nothing to transition
		return nil, nil
	}

	marker := &Marker{}
	for i, line := range lines {
		if i == 0 {
			marker.CommitID = strings.TrimSpace(line)
		} else if i == 1 {
			marker.Version = strings.TrimSpace(line)
		}
	}

	data, err := json.Marshal(marker)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal marker data")
	}

	err = fileutils.WriteFile(s.markerFile(), data)
	if err != nil {
		return nil, errs.Wrap(err, "could not set completion marker")
	}

	return marker, nil
}

// MarkInstallationComplete writes the installation complete marker to the runtime directory
func (s *Store) MarkInstallationComplete(commitID strfmt.UUID, namespace string) error {
	markerFile := s.markerFile()
	markerDir := filepath.Dir(markerFile)
	err := fileutils.MkdirUnlessExists(markerDir)
	if err != nil {
		return errs.Wrap(err, "could not create completion marker directory")
	}

	data, err := json.Marshal(Marker{commitID.String(), namespace, constants.Version})
	if err != nil {
		return errs.Wrap(err, "Could not marshal marker data")
	}

	err = fileutils.WriteFile(markerFile, data)
	if err != nil {
		return errs.Wrap(err, "could not set completion marker")
	}

	return nil
}

func (s *Store) CommitID() (string, error) {
	marker, err := s.parseMarker()
	if err != nil {
		return "", errs.Wrap(err, "Could not parse marker file")
	}

	return marker.CommitID, nil
}

func (s *Store) Namespace() (string, error) {
	marker, err := s.parseMarker()
	if err != nil {
		return "", errs.Wrap(err, "Could not parse marker file")
	}

	return marker.Namespace, nil
}
