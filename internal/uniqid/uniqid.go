package uniqid

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/go-openapi/strfmt"
)

// TODO: wrap errors

const (
	fileName = "activestate.dat"
)

// UniqID manages the storage and retrieval of a unique id.
type UniqID struct {
	ID strfmt.UUID
}

// New retrieves or creates a new unique id.
func New() (*UniqID, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	idText, err := uniqIDText(filepath.Join(dir, fileName))
	if err != nil {
		return nil, err
	}

	// TODO: convert to uuid or err
	_ = idText
	var id strfmt.UUID

	return &UniqID{ID: id}, nil
}

// String implements fmt.Stringer.
func (u *UniqID) String() string {
	return u.ID.String()
}

func uniqIDText(filepath string) (string, error) {
	data, err := fileutils.ReadFile(filepath)
	if err == nil {
		return string(data), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		// TODO: create new uuid
		uniqID := "test"

		if err := fileutils.WriteFile(filepath, []byte(uniqID)); err != nil {
			return "", err
		}

		return uniqID, nil
	}

	return "", err
}
