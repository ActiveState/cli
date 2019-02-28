package python

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	datadir     string
	artifact    *artifact.Artifact
	packagePath string
}

// Language - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Language() string {
	if v.artifact == nil {
		return "python3"
	}
	return strings.ToLower(v.artifact.Meta.Name)
}

// DataDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) DataDir() string {
	return v.datadir
}

// SetDataDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) SetDataDir(path string) {
	v.datadir = path
}

// Artifact - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Artifact() *artifact.Artifact {
	return v.artifact
}

// SetArtifact - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) SetArtifact(artf *artifact.Artifact) {
	v.artifact = artf
}

// LoadArtifact - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) LoadArtifact(artf *artifact.Artifact) *failures.Failure {
	switch artf.Meta.Type {
	case "language":
		return v.loadLanguage(artf)
	case "package":
		return v.loadPackage(artf)
	default:
		return failures.FailUser.New("err_artifact_not_supported", artf.Meta.Type)
	}
}

// WorkingDirectory - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) WorkingDirectory() string {
	return ""
}

func (v *VirtualEnvironment) loadLanguage(artf *artifact.Artifact) *failures.Failure {
	err := os.Symlink(filepath.Dir(artf.Path), filepath.Join(v.DataDir(), "language"))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return nil
}

func (v *VirtualEnvironment) loadPackage(artf *artifact.Artifact) *failures.Failure {
	if err := fileutils.Mkdir(v.datadir, "lib"); err != nil {
		return failures.FailIO.Wrap(err)
	}

	artfPath := filepath.Dir(artf.Path)
	err := filepath.Walk(artfPath, func(subpath string, f os.FileInfo, err error) error {
		subpath = strings.TrimPrefix(subpath, artfPath)
		if subpath == "" {
			return nil
		}
		var target string
		if runtime.GOOS == "windows" {
			target = filepath.Join(v.DataDir(), "language", "Lib", "site-packages", artf.Meta.Name, subpath)
		} else {
			langLibPath := v.getPackageFolder(filepath.Join(v.DataDir(), "language", "lib"))
			target = filepath.Join(langLibPath, "site-packages", artf.Meta.Name, subpath)
		}
		if fileutils.PathExists(target) {
			return nil
		}

		if err := fileutils.Mkdir(filepath.Dir(target), "lib"); err != nil {
			return failures.FailIO.Wrap(err)
		}
		return os.Symlink(filepath.Join(artfPath, subpath), target)
	})

	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return nil
}

func (v *VirtualEnvironment) getPackageFolder(path string) string {
	if v.packagePath != "" {
		return v.packagePath
	}

	matches, err := filepath.Glob(filepath.Join(path, "python*"))
	if err != nil {
		return ""
	}
	if len(matches) == 0 {
		return ""
	}

	v.packagePath = matches[0]
	return v.packagePath
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	if err := fileutils.Mkdir(v.datadir, "bin"); err != nil {
		return err
	}
	return fileutils.Mkdir(v.datadir, "lib")
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	env := map[string]string{}
	if distPath, found := v.pathToAnyDistribution(); found {
		logging.Debug("found distribution '%s'", distPath)
		env["PATH"] = path.Join(distPath, "bin")
	}
	return env
}

// pathToAnyDistribution will return the path to the first distribution dir found.
func (v *VirtualEnvironment) pathToAnyDistribution() (string, bool) {
	distsDirPath := path.Join(v.datadir, constants.ActivePythonDistsDir)
	if !fileutils.DirExists(distsDirPath) {
		logging.Debug("distributions dir '%s' does not exist", distsDirPath)
		return "", false
	}

	distsDir, err := os.Open(distsDirPath)
	if err != nil {
		logging.Error("accessing distributions dir '%s': %v", distsDirPath, err)
		return "", false
	}
	defer distsDir.Close()

	// read one directory name
	distDirNames, err := distsDir.Readdirnames(1)
	if err != nil {
		if err == io.EOF {
			logging.Debug("no distributions found in '%s'", distsDirPath)
		} else {
			logging.Error("reading dir names from distributions dir '%s': %v", distsDirPath, err)
		}
		return "", false
	}

	return path.Join(distsDirPath, distDirNames[0]), true
}
