package vars

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type projectable interface {
	Owner() string
	Name() string
	NamespaceString() string
	CommitID() string
	BranchName() string
	Path() string
	URL() string
}

type Project struct {
	Namespace string `vars:"NamespacePrefix"`
	Name      string
	Owner     string
	URL       string `vars:"Url"`
	Commit    string
	Branch    string
	Path      string
}

type OS struct {
	Name         string
	Version      *sysinfo.OSVersionInfo
	Architecture string
}

type User struct {
	Name  string
	Email string
}

type Mixin struct {
	User *User
}

type Vars struct {
	Project *Project
	OS      *OS
	Shell   string
	Mixin   *Mixin
}

func New(auth *authentication.Auth, pj projectable, subshellName string) *Vars {
	var (
		proj = &Project{}
	)
	if !p.IsNil(pj) {
		proj.Namespace = pj.NamespaceString()
		proj.Owner = pj.Owner()
		proj.Name = pj.Name()
		proj.URL = pj.URL()
		proj.Commit = pj.CommitID()
		proj.Branch = pj.BranchName()
		proj.Path = pj.Path()
		if proj.Path != "" {
			proj.Path = filepath.Dir(proj.Path)
		}
	}

	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		multilog.Error("Could not detect OSVersion: %v", err)
	}

	os := &OS{
		Name:         sysinfo.OS().String(),
		Version:      osVersion,
		Architecture: sysinfo.Architecture().String(),
	}

	return &Vars{
		Project: proj,
		OS:      os,
		Shell:   subshellName,
		Mixin:   mixin,
	}
}
