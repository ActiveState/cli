// Package vars provides a single type expressing the data accessible by the
// activestate.yaml for conditionals and variable expansions.
//
// The structure should not grow beyond a depth of 3. That is, .OS.Version.Major
// is fine, but .OS.Version.Major.Something is not. External (leaf) nodes must
// be able to resolve to a string using `fmt.Sprintf("%v")`. Keep in mind that
// the Vars type itself is depth 0, so it does not count for depth, and is
// represented in the activestate.yaml as either the first `.` or the `$`.
//
// Nodes at depth 1 may be a function, but the return value must also resolve
// to a string using `fmt.Sprintf("%v")`. A second return value of `error` is
// allowed. For variable expansion, a non-function node may be tagged as a
// function (asFunc) so that it must be called using parenthesis
// (`$project.name()`).
//
// Path nodes should be tagged (isPath) so that bashification of the path is
// applied when necessary.
package vars

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type Project struct {
	Namespace string `expand:",asFunc"`
	Name      string `expand:",asFunc"`
	Owner     string `expand:",asFunc"`
	Url       string `expand:",asFunc"`
	Commit    string `expand:",asFunc"`
	Branch    string `expand:",asFunc"`
	Path      string `expand:",asFunc;isPath"`

	// legacy fields
	NamespacePrefix string
}

func NewProject(pj *project.Project) *Project {
	p := &Project{}
	p.Update(pj)
	return p
}

func (p *Project) Update(pj *project.Project) {
	p.Namespace = pj.NamespaceString()
	p.Name = pj.Name()
	p.Owner = pj.Owner()
	p.Url = pj.URL()
	p.Commit = pj.CommitID()
	p.Branch = pj.BranchName()
	p.Path = pj.Path()
	if p.Path != "" {
		p.Path = filepath.Dir(p.Path)
	}
	p.NamespacePrefix = pj.NamespaceString()
}

type OSVersion struct {
	Name    string
	Version string
	Major   int
	Minor   int
	Micro   int
}

type OS struct {
	Name         string
	Version      OSVersion
	Architecture string
}

func NewOS(osVersion *sysinfo.OSVersionInfo) *OS {
	return &OS{
		Name: sysinfo.OS().String(),
		Version: OSVersion{
			Name:    osVersion.Name,
			Version: osVersion.Version,
			Major:   osVersion.Major,
			Minor:   osVersion.Minor,
			Micro:   osVersion.Micro,
		},
		Architecture: sysinfo.Architecture().String(),
	}
}

type User struct {
	Name  string
	Email string
	JWT   string
}

type Mixin struct {
	auth *authentication.Auth
	User *User
}

func NewMixin(auth *authentication.Auth) *Mixin {
	return &Mixin{
		auth: auth,
		User: &User{
			Name:  auth.WhoAmI(),
			Email: auth.Email(),
			JWT:   auth.BearerToken(),
		},
	}
}

type Vars struct {
	Project *Project
	OS      *OS
	Shell   string
	Mixin   func() *Mixin
}

func New(auth *authentication.Auth, pj *project.Project, subshellName string) *Vars {
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		multilog.Error("Could not detect OSVersion: %v", err)
	}

	return &Vars{
		Project: NewProject(pj),
		OS:      NewOS(osVersion),
		Shell:   subshellName,
		Mixin:   func() *Mixin { return NewMixin(auth) },
	}
}
