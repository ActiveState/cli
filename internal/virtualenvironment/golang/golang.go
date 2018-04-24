package golang

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/scm"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/projectfile"
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

// WorkingDirectory - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) WorkingDirectory() string {
	return filepath.Join(v.DataDir(), "src", v.namespace())
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

	project := projectfile.Get()

	namespace := v.namespace()

	fail := fileutils.Mkdir(filepath.Join(v.DataDir(), "src", filepath.Dir(namespace)))
	if fail != nil {
		return fail
	}

	err := os.Symlink(filepath.Dir(project.Path()), filepath.Join(v.DataDir(), "src", namespace))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return fileutils.Mkdir(v.DataDir(), "bin")
}

// namespace retrieves the namespace to use for the current venv
func (v *VirtualEnvironment) namespace() string {
	project := projectfile.Get()
	if project.Namespace != "" {
		return project.Namespace
	}

	projectPath := filepath.Dir(project.Path())
	scmm := scm.FromPath(projectPath)
	if scmm != nil {
		uri := scmm.URI()

		if uri[0:4] == "git@" {
			uri = strings.Replace(uri, ":", "/", 1)
			uri = strings.Replace(uri, "git@", "http://", 1)
		}
		uri = strings.Replace(uri, ".git", "", 1)

		url, err := url.Parse(uri)
		if err == nil {
			return path.Join(url.Hostname(), url.Path)
		}
	}

	return path.Join(constants.DefaultNamespaceDomain, project.Owner, project.Name)
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	return map[string]string{
		"GOPATH": v.DataDir(),
		"GOBIN":  filepath.Join(v.DataDir(), "bin"),
		"GOROOT": filepath.Join(v.DataDir(), "language"),
		"PATH":   filepath.Join(v.DataDir(), "language", "bin"),
	}
}
