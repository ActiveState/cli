package virtualenvironment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/mholt/archiver"
	"github.com/mitchellh/hashstructure"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/virtualenvironment/python"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
)

// VirtualEnvironmenter defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type VirtualEnvironmenter interface {
	// Activate the given virtualenvironment
	Activate() error

	// Language returns the language name
	Language() string

	// LanguageMeta holds the *projectfile.Language for this venv
	LanguageMeta() *projectfile.Language

	// SetLanguageMeta sets the language meta
	SetLanguageMeta(*projectfile.Language)

	// DataDir returns the configured data dir for this venv
	DataDir() string

	// SetDataDir sets the configured data for this venv
	SetDataDir(string)

	// LoadLanguageFromPath should load the given language into the venv via symlinks
	LoadLanguageFromPath(string) error

	// LoadLanguageFromPath should load the given package into the venv via symlinks
	LoadPackageFromPath(string, *projectfile.Package) error
}

type artefactHashable struct {
	Name    string
	Version string
	Build   map[string]string
}

var venvs = make(map[string]VirtualEnvironmenter)

// Activate the virtual environment
func Activate() error {
	logging.Debug("Activating Virtual Environment")

	project := projectfile.Get()

	if project.Variables != nil {
		for _, variable := range project.Variables {
			// TODO: if !constraints.IsConstrained(variable.Constraints, project)
			os.Setenv(variable.Name, variable.Value)
		}
	}

	err := createFolderStructure()
	if err != nil {
		return err
	}

	for _, language := range project.Languages {
		_, err := GetEnv(&language)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetEnv returns an environment for the given project and language, this will initialize the virtual directory structure
// and set up the necessary environment variables if the venv wasnt already initialized, otherwise it will just return
// the venv struct
func GetEnv(language *projectfile.Language) (VirtualEnvironmenter, error) {
	switch language.Name {
	case "Python":
		// TODO: if !constraints.IsConstrained(language.Constraints, project)
		hash := getHashFromLanguage(language)
		if _, ok := venvs[hash]; ok {
			return venvs[hash], nil
		}

		venv := &python.VirtualEnvironment{}
		err := ActivateLanguageVenv(language, venv)

		if err != nil {
			return nil, err
		}

		venvs[hash] = venv

		return venv, nil
	default:
		var T = locale.T
		return nil, failures.FailUser.New(T("warning_language_not_yet_supported", map[string]interface{}{
			"Language": language.Name,
		}))
	}
}

// ActivateLanguageVenv activates the virtual environment for the given language
func ActivateLanguageVenv(language *projectfile.Language, venv VirtualEnvironmenter) error {
	project := projectfile.Get()
	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "virtual", project.Owner, project.Name, language.Name, language.Version)

	venv.SetLanguageMeta(language)
	venv.SetDataDir(datadir)

	err := loadLanguage(language, venv)

	if err != nil {
		return err
	}

	for _, pkg := range language.Packages {
		// TODO: if !constraints.IsConstrained(pkg.Constraints, project)
		err = loadPackage(language, &pkg, venv)

		if err != nil {
			return err
		}
	}

	return venv.Activate()
}

func loadLanguage(language *projectfile.Language, venv VirtualEnvironmenter) error {
	path, err := obtainLanguage(language)

	if err != nil {
		return err
	}

	logging.Debug("Loading Language %s", language.Name)

	return venv.LoadLanguageFromPath(path)
}

func getHashFromLanguage(language *projectfile.Language) string {
	hashable := artefactHashable{Name: language.Name, Version: language.Version, Build: language.Build}
	hash, _ := hashstructure.Hash(hashable, nil)
	return fmt.Sprintf("%d", hash)
}

func obtainLanguage(language *projectfile.Language) (string, error) {
	root, err := environment.GetRootPath()

	if err != nil {
		return "", err
	}

	datadir := config.GetDataDir()

	path := filepath.Join(datadir, "languages", language.Name, getHashFromLanguage(language))

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}

	logging.Debug("Obtaining Language %s", language.Name)

	// Black box stuff that needs to be replaced with API calls
	input := filepath.Join(root, "test", "builder", strings.ToLower(language.Name), language.Version+".tar.gz")
	err = archiver.TarGz.Open(input, path)

	if err != nil {
		return "", err
	}

	return path, nil
}

func loadPackage(language *projectfile.Language, pkg *projectfile.Package, venv VirtualEnvironmenter) error {
	path, err := obtainPackage(language, pkg)

	if err != nil {
		return err
	}

	logging.Debug("Loading Package %s", pkg.Name)

	return venv.LoadPackageFromPath(path, pkg)
}

func getHashFromPackage(pkg *projectfile.Package) string {
	hashable := artefactHashable{Name: pkg.Name, Version: pkg.Version, Build: pkg.Build}
	hash, _ := hashstructure.Hash(hashable, nil)
	return fmt.Sprintf("%d", hash)
}

func obtainPackage(language *projectfile.Language, pkg *projectfile.Package) (string, error) {
	root, err := environment.GetRootPath()

	if err != nil {
		return "", err
	}

	datadir := config.GetDataDir()

	path := filepath.Join(datadir, "packages", pkg.Name, getHashFromPackage(pkg))

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}

	logging.Debug("Obtaining Package %s", pkg.Name)

	// Black box stuff that needs to be replaced with API calls
	input := filepath.Join(
		root, "test", "builder",
		strings.ToLower(language.Name), strings.ToLower(language.Version),
		strings.ToLower(pkg.Name), pkg.Version+".tar.gz")
	err = archiver.TarGz.Open(input, path)

	if err != nil {
		return "", err
	}

	return path, nil
}

// small helper function to create a directory if it doesnt already exist
func mkdir(parent string, subpath ...string) error {
	path := filepath.Join(subpath...)
	path = filepath.Join(parent, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}

func createFolderStructure() error {
	project := projectfile.Get()
	datadir := config.GetDataDir()

	if err := mkdir(datadir, "packages"); err != nil {
		return err
	}

	if err := mkdir(datadir, "languages"); err != nil {
		return err
	}

	os.RemoveAll(filepath.Join(datadir, "virtual", project.Owner, project.Name))

	for _, language := range project.Languages {
		mkdir(datadir, "virtual", project.Owner, project.Name, language.Name, language.Version)
	}

	return nil
}
