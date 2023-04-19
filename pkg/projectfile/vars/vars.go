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

func NewProject(pj projectDataProvider) *Project {
	var (
		project = &Project{}
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

func New(auth *authentication.Auth, project *Project, subshellName string) *Vars {
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		multilog.Error("Could not detect OSVersion: %v", err)
	}

	return &Vars{
		Project: project,
		OS:      NewOS(osVersion),
		Shell:   subshellName,
		Mixin:   func() *Mixin { return NewMixin(auth) },
	}
}
