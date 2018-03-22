package golang

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/artifact"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/fileutils"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	datadir  string
	artifact *artifact.Artifact
}

// Language - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Language() string {
	return "go"
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
		return failures.FailUser.New("err_language_not_supported", artf.Meta.Name)
	}
}

func (v *VirtualEnvironment) loadLanguage(artf *artifact.Artifact) *failures.Failure {
	err := os.Symlink(filepath.Dir(artf.Path), filepath.Join(v.DataDir(), "language"))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return nil
}

func (v *VirtualEnvironment) loadPackage(artf *artifact.Artifact) *failures.Failure {
	if err := fileutils.Mkdir(v.DataDir(), "src", filepath.Dir(artf.Meta.Name)); err != nil {
		return failures.FailIO.Wrap(err)
	}

	err := os.Symlink(filepath.Dir(artf.Path), filepath.Join(v.DataDir(), "src", artf.Meta.Name))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return nil
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	logging.Debug("Activating Go venv")

	return fileutils.Mkdir(v.datadir, "bin")
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	return map[string]string{
		"GOPATH": v.datadir,
		"GOBIN":  filepath.Join(v.datadir, "bin"),
		"GOROOT": filepath.Join(v.DataDir(), "language"),
		"PATH":   filepath.Join(v.DataDir(), "language", "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
}
