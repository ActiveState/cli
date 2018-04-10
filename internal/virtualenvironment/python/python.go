package python

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	datadir      string
	languageMeta *projectfile.Language
}

// Language - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Language() string {
	return "Python"
}

// DataDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) DataDir() string {
	return v.datadir
}

// SetDataDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) SetDataDir(path string) {
	v.datadir = path
}

// LanguageMeta - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) LanguageMeta() *projectfile.Language {
	return v.languageMeta
}

// SetLanguageMeta - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) SetLanguageMeta(language *projectfile.Language) {
	v.languageMeta = language
}

// LoadLanguageFromPath - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) LoadLanguageFromPath(path string) error {
	err := os.Symlink(path, filepath.Join(v.DataDir(), "language"))
	if err != nil {
		logging.Error(err.Error())
		return failures.FailIO.New(locale.T("error_could_not_make_symlink"))
	}
	return nil
}

// LoadPackageFromPath - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) LoadPackageFromPath(path string, pkg *projectfile.Package) error {
	if err := mkdir(v.datadir, "lib"); err != nil {
		return err
	}

	return filepath.Walk(path, func(subpath string, f os.FileInfo, err error) error {
		subpath = strings.TrimPrefix(subpath, path)
		if subpath == "" {
			return nil
		}
		return os.Symlink(filepath.Join(path, subpath), filepath.Join(v.DataDir(), "lib", subpath))
	})
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() error {
	if err := mkdir(v.datadir, "bin"); err != nil {
		return err
	}
	if err := mkdir(v.datadir, "lib"); err != nil {
		return err
	}

	logging.Debug("Setting up Python env variables")

	os.Setenv("PYTHONPATH", filepath.Join(v.datadir, "lib"))
	os.Setenv("PATH", filepath.Join(v.datadir, "language", "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	os.Setenv("PATH", filepath.Join(v.datadir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))

	return nil
}

// small helper function to create a directory if it doesnt already exist
func mkdir(parent string, subpath ...string) error {
	path := filepath.Join(subpath...)
	path = filepath.Join(parent, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModePerm)
	}
	return nil
}
