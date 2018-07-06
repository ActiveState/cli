package python

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	datadir      string
	artifact     *artifact.Artifact
	packagePaths map[string]string
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
	if v.packagePaths == nil {
		v.packagePaths = make(map[string]string)
	}
	if v.packagePaths[path+v.Language()] != "" {
		return v.packagePaths[path+v.Language()]
	}

	matches, err := filepath.Glob(filepath.Join(path, "python*"))
	if err != nil {
		return ""
	}
	if len(matches) == 0 {
		return ""
	}

	v.packagePaths[path+v.Language()] = matches[0]
	return v.packagePaths[path+v.Language()]
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
	path := filepath.Join(v.datadir, "bin")
	// Windows Python directory does NOT contain a `bin` directory for
	// binaries
	if runtime.GOOS == "windows" {
		path = filepath.Join(v.datadir, "language") + string(os.PathListSeparator) + path
	} else {
		path = filepath.Join(v.datadir, "language", "bin") + string(os.PathListSeparator) + path
	}

	return map[string]string{
		"PYTHONPATH": filepath.Join(v.datadir, "lib"),
		"PATH":       path,
	}
}
