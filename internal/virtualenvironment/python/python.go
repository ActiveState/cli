package python

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	datadir  string
	artifact *artifact.Artifact
}

// Language - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Language() string {
	return "python"
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
	case "package":
		return v.loadPackage(artf)
	default:
		return failures.FailUser.New("err_language_not_supported", artf.Meta.Name)
	}
}

// WorkingDirectory - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) WorkingDirectory() string {
	return ""
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
		target := filepath.Join(v.DataDir(), "lib", filepath.Base(artfPath), subpath)
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

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	if err := fileutils.Mkdir(v.datadir, "bin"); err != nil {
		return err
	}
	return fileutils.Mkdir(v.datadir, "lib")
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	path := filepath.Join(v.datadir, "language", "bin") + string(os.PathListSeparator) + os.Getenv("PATH")
	path = filepath.Join(v.datadir, "bin") + string(os.PathListSeparator) + path
	return map[string]string{
		"PYTHONPATH": filepath.Join(v.datadir, "lib"),
		"PATH":       path,
	}
}
