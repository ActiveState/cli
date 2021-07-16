package uniqid

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/google/uuid"
)

// DirLocation represents tokens to indicate where uniqid file should be
// located.
type DirLocation int

// DirLocation enums.
const (
	InHome DirLocation = iota
	InTmp
)

const (
	fileName = "uniqid"
	asDir    = "activestate_dat"
	tmpSub   = "activestate_uniqid"
)

// UniqID manages the storage and retrieval of a unique id.
type UniqID struct {
	ID uuid.UUID
}

// New retrieves or creates a new unique id.
func New(in DirLocation) (*UniqID, error) {
	dir, err := storageDirectory(in)
	if err != nil {
		return nil, errs.Wrap(err, "cannot determine uniqid storage directory")
	}

	id, err := uniqueID(filepath.Join(dir, fileName))
	if err != nil {
		return nil, errs.Wrap(err, "cannot obtain uniqid")
	}

	return &UniqID{ID: id}, nil
}

// String implements fmt.Stringer.
func (u *UniqID) String() string {
	return u.ID.String()
}

func uniqueID(filepath string) (uuid.UUID, error) {
	data, err := fileutils.ReadFile(filepath)
	if err == nil {
		id, err := uuid.FromBytes(data)
		if err == nil {
			return id, nil
		}
		err = os.ErrNotExist // signal to clobber existing file
	}

	if errors.Is(err, os.ErrNotExist) {
		id := uuid.New()

		if err := fileutils.WriteFile(filepath, id[:]); err != nil {
			return uuid.UUID{}, errs.Wrap(err, "cannot write uniqid file")
		}

		return id, nil
	}

	return uuid.UUID{}, errs.Wrap(err, "cannot get existing, nor create new, uniqid")
}

// ErrUnsupportedOS indicates that an unsupported OS tried to store a uniqid as
// a file.
var ErrUnsupportedOS = errors.New("unsupported uniqid os")

func storageDirectory(location DirLocation) (string, error) {
	var dir string
	switch location {
	case InTmp:
		dir = filepath.Join(os.TempDir(), tmpSub)

	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errs.Wrap(err, "cannot get home dir for uniqid file")
		}
		dir = home
	}

	var subdir string
	switch runtime.GOOS {
	case "darwin":
		subdir = "Library/Application Support"
	case "linux":
		subdir = ".local/share"
	case "windows":
		subdir = "AppData"
	default:
		return "", ErrUnsupportedOS
	}

	return filepath.Join(dir, subdir, asDir), nil
}
