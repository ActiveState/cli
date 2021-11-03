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

// BaseDirLocation facilitates base directory location option enums.
type BaseDirLocation int

// BaseDirLocation enums define the base directory location options.
const (
	InHome BaseDirLocation = iota
	InTmp
)

const (
	fileName         = "uniqid"
	legacyPersistDir = "activestate/persist"
	persistDir       = "activestate_persist"
)

// UniqID manages the storage and retrieval of a unique id.
type UniqID struct {
	ID uuid.UUID
}

// New retrieves or creates a new unique id.
func New(base BaseDirLocation) (*UniqID, error) {
	dir, err := storageDirectory(base, false)
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
	// For a transitionary period where the old persist directory may exist on
	// Windows we want to move the uniqid file to a better location.
	// This code should be removed after some time.
	if !fileExists(filepath) {
		err := moveLegacyFile(filepath)
		if err != nil {
			return uuid.UUID{}, errs.Wrap(err, "Could not move legacy uniqid file")
		}
	}

	data, err := readFile(filepath)
	if errors.Is(err, os.ErrNotExist) {
		id := uuid.New()

		if err := writeFile(filepath, id[:]); err != nil {
			return uuid.UUID{}, errs.Wrap(err, "cannot write uniqid file")
		}

		return id, nil
	}
	if err != nil {
		return uuid.UUID{}, errs.Wrap(err, "Could not read uniqid file")
	}

	id, err := uuid.FromBytes(data)
	if err != nil {
		return uuid.UUID{}, errs.Wrap(err, "Could not create new UUID from uniqid file data")
	}

	return id, nil
}

func moveLegacyFile(destination string) error {
	legacyDir, err := storageDirectory(InHome, true)
	if err != nil {
		return errs.Wrap(err, "Could not get legacy storage directory")
	}

	// If the legacy file does not not exist there is nothing to move
	if !fileExists(filepath.Join(legacyDir, fileName)) {
		return nil
	}

	destinationDir := filepath.Dir(destination)
	err = mkdir(destinationDir)
	if err != nil {
		return errs.Wrap(err, "Could not create new persist directory")
	}

	err = moveAllFiles(legacyDir, destinationDir)
	if err != nil {
		return errs.Wrap(err, "Could not move legacy uniqid file")
	}

	// The legacy directory is a sub directory, we want to remove the parent
	err = os.RemoveAll(filepath.Dir(legacyDir))
	if err != nil {
		return errs.Wrap(err, "Could not remove legacy uniqid dir")
	}

	return nil
}

// ErrUnsupportedOS indicates that an unsupported OS tried to store a uniqid as
// a file.
var ErrUnsupportedOS = errors.New("unsupported uniqid os")

func storageDirectory(base BaseDirLocation, legacy bool) (string, error) {
	var dir string
	switch base {
	case InTmp:
		dir = filepath.Join(os.TempDir(), persistDir)

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
		if legacy {
			return filepath.Join(dir, "AppData", legacyPersistDir), nil
		}
		subdir = "AppData/Local"
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

// moveAllFiles will move all of the files/dirs within one directory to another directory. Both directories
// must already exist.
func moveAllFiles(fromPath, toPath string) error {
	if !dirExists(fromPath) {
		return errs.New("Expected '%s' to be a valid directory", fromPath)
	} else if !dirExists(toPath) {
		errs.New("Expected '%s' to be a valid directory", toPath)
	}

	// read all child files and dirs
	dir, err := os.Open(fromPath)
	if err != nil {
		return errs.Wrap(err, "os.Open %s failed", fromPath)
	}
	fileInfos, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		return errs.Wrap(err, "dir.Readdir %s failed", fromPath)
	}

	// any found files and dirs
	for _, fileInfo := range fileInfos {
		fromPath := filepath.Join(fromPath, fileInfo.Name())
		toPath := filepath.Join(toPath, fileInfo.Name())
		err := os.Rename(fromPath, toPath)
		if err != nil {
			return errs.Wrap(err, "os.Rename %s:%s failed", fromPath, toPath)
		}
	}
	return nil
}
