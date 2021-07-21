package uniqid

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
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
	fileName   = "uniqid"
	persistDir = "activestate/persist"
	tmpSubDir  = "activestate_persist"
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
	data, err := readFile(filepath)
	if err == nil {
		id, err := uuid.FromBytes(data)
		if err == nil {
			return id, nil
		}
		err = os.ErrNotExist // signal to clobber existing file
	}

	if errors.Is(err, os.ErrNotExist) {
		id := uuid.New()

		if err := writeFile(filepath, id[:]); err != nil {
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
		dir = filepath.Join(os.TempDir(), tmpSubDir)

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

	return filepath.Join(dir, subdir, persistDir), nil
}

// The following is copied from fileutils to avoid cyclical importing. As
// global usage in the code is minimized, or logging is removed from fileutils,
// this may be removed.

// readFile reads the content of a file
func readFile(filePath string) ([]byte, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errs.Wrap(err, "ioutil.ReadFile %s failed", filePath)
	}
	return b, nil
}

// dirExists checks if the given directory exists
func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := fi.Mode()
	return mode.IsDir()
}

// mkdir is a small helper function to create a directory if it doesnt already exist
func mkdir(path string, subpath ...string) error {
	if len(subpath) > 0 {
		subpathStr := filepath.Join(subpath...)
		path = filepath.Join(path, subpathStr)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("MkdirAll failed for path: %s", path))
		}
	}
	return nil
}

// mkdirUnlessExists will make the directory structure if it doesn't already exists
func mkdirUnlessExists(path string) error {
	if dirExists(path) {
		return nil
	}
	return mkdir(path)
}

// fileExists checks if the given file (not folder) exists
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := fi.Mode()
	return mode.IsRegular()
}

// writeFile writes data to a file, if it exists it is overwritten, if it doesn't exist it is created and data is written
func writeFile(filePath string, data []byte) error {
	err := mkdirUnlessExists(filepath.Dir(filePath))
	if err != nil {
		return err
	}

	// make the target file temporarily writable
	fileExists := fileExists(filePath)
	if fileExists {
		stat, _ := os.Stat(filePath)
		if err := os.Chmod(filePath, 0644); err != nil {
			return errs.Wrap(err, "os.Chmod %s failed", filePath)
		}
		defer os.Chmod(filePath, stat.Mode().Perm())
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		if !fileExists {
			target := filepath.Dir(filePath)
			err = fmt.Errorf("access to target %q is denied", target)
		}
		return errs.Wrap(err, "os.OpenFile %s failed", filePath)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return errs.Wrap(err, "file.Write %s failed", filePath)
	}
	return nil
}
