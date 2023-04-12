package vars

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type projectDataProvider interface {
	Owner() string
	Name() string
	NamespaceString() string
	CommitID() string
	BranchName() string
	Path() string
	URL() string
}

type ProjectData struct {
	Namespace string
	Name      string
	Owner     string
	Url       string
	Commit    string
	Branch    string
	Path      string

	// legacy fields
	NamespacePrefix string
}

func NewProject(pj projectDataProvider) *ProjectData {
	var (
		project = &ProjectData{}
	)
	if !p.IsNil(pj) {
		project.Namespace = pj.NamespaceString()
		project.Name = pj.Name()
		project.Owner = pj.Owner()
		project.Url = pj.URL()
		project.Commit = pj.CommitID()
		project.Branch = pj.BranchName()
		project.Path = pj.Path()
		if project.Path != "" {
			project.Path = filepath.Dir(project.Path)
		}

		project.NamespacePrefix = pj.NamespaceString()
	}

	return project
}

type OS struct {
	Name         string
	Version      *sysinfo.OSVersionInfo
	Architecture string
}

func NewOS(osVersion *sysinfo.OSVersionInfo) *OS {
	return &OS{
		Name:         sysinfo.OS().String(),
		Version:      osVersion,
		Architecture: sysinfo.Architecture().String(),
	}
}

type User struct {
	Name  string
	Email string
}

type Mixin struct {
	User    *User
	Example string
}

type Vars struct {
	Project *ProjectData
	OS      *OS
	Shell   string
	Mixin   func() *Mixin
}

func New(auth *authentication.Auth, project *ProjectData, subshellName string) *Vars {
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		multilog.Error("Could not detect OSVersion: %v", err)
	}

	mixin := func() *Mixin {
		return &Mixin{
			User: &User{
				Name:  "NAME",
				Email: "EMAIL",
			},
			Example: "EXAMPLE",
		}
	}
	return &Vars{
		Project: project,
		OS:      NewOS(osVersion),
		Shell:   subshellName,
		Mixin:   mixin,
	}
}
