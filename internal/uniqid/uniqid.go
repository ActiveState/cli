package uniqid

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

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
	fileName   = "uniqid"
	persistDir = "activestate_persist"
)

// UniqID manages the storage and retrieval of a unique id.
type UniqID struct {
	ID uuid.UUID
}

// New retrieves or creates a new unique id.
func New(base BaseDirLocation) (*UniqID, error) {
	dir, err := storageDirectory(base)
	if err != nil {
		return nil, fmt.Errorf("cannot determine uniqid storage directory: %w", err)
	}

	id, err := uniqueID(filepath.Join(dir, fileName))
	if err != nil {
		return nil, fmt.Errorf("cannot obtain uniqid: %w", err)
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
		err := moveUniqidFile(filepath)
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("Could not move legacy uniqid file: %w", err)
		}
	}

	data, err := readFile(filepath)
	if errors.Is(err, os.ErrNotExist) {
		id := uuid.New()

		if err := writeFile(filepath, id[:]); err != nil {
			return uuid.UUID{}, fmt.Errorf("cannot write uniqid file: %w", err)
		}

		return id, nil
	}
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("Could not read uniqid file: %w", err)
	}

	id, err := uuid.FromBytes(data)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("Could not create new UUID from uniqid file data: %w", err)
	}

	return id, nil
}

// ErrUnsupportedOS indicates that an unsupported OS tried to store a uniqid as
// a file.
var ErrUnsupportedOS = errors.New("unsupported uniqid os")

func storageDirectory(base BaseDirLocation) (string, error) {
	var dir string
	switch base {
	case InTmp:
		dir = filepath.Join(os.TempDir(), persistDir)

	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot get home dir for uniqid file: %w", err)
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
		return nil, fmt.Errorf("ioutil.ReadFile %s failed: %w", filePath, err)
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
			return fmt.Errorf("MkdirAll failed for path: %s: %w", path, err)
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
			return fmt.Errorf("os.Chmod %s failed: %w", filePath, err)
		}
		defer os.Chmod(filePath, stat.Mode().Perm())
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		if !fileExists {
			target := filepath.Dir(filePath)
			err = fmt.Errorf("access to target %q is denied", target)
		}
		return fmt.Errorf("os.OpenFile %s failed: %w", filePath, err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("file.Write %s failed: %w", filePath, err)
	}
	return nil
}

// copyFile copies a file from one location to another
func copyFile(src, target string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open %s failed: %w", src, err)
	}
	defer in.Close()

	// Create target directory if it doesn't exist
	dir := filepath.Dir(target)
	err = mkdirUnlessExists(dir)
	if err != nil {
		return err
	}

	// Create target file
	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("os.Create %s failed: %w", target, err)
	}
	defer out.Close()

	// Copy bytes to target file
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("io.Copy failed: %w", err)
	}
	err = out.Close()
	if err != nil {
		return fmt.Errorf("out.Close failed: %w", err)
	}
	return nil
}
