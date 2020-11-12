package projects

import (
	"sort"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Holds a union of project and organization parameters.
type projectWithOrg struct {
	Name           string   `json:"name"`
	Organization   string   `json:"organization"`
	LocalCheckouts []string `json:"local_checkouts,omitempty" locale:"local_checkouts,Local Checkouts" opts:"emptyNil,singleLine"`
}

type configGetter interface {
	GetStringMapStringSlice(key string) map[string][]string
}

// Params are command line parameters
type Params struct {
	Local bool // Whether to show locally checked out projects only
}

type Projects struct {
	auth   *authentication.Auth
	out    output.Outputer
	config configGetter
}

type primeable interface {
	primer.Auther
	primer.Outputer
}

func NewParams() *Params {
	return &Params{Local: false}
}

func NewProjects(prime primeable, config configGetter) *Projects {
	return newProjects(prime.Auth(), prime.Output(), config)
}

func newProjects(auth *authentication.Auth, out output.Outputer, config configGetter) *Projects {
	return &Projects{
		auth,
		out,
		config,
	}
}

func (r *Projects) Run(params *Params) error {
	projectfile.CleanProjectMapping()
	localProjects := r.config.GetStringMapStringSlice(projectfile.LocalProjectsConfigKey)

	projects := []projectWithOrg{}
	for namespace, checkouts := range localProjects {
		ns, fail := project.ParseNamespace(namespace)
		if fail != nil {
			logging.Error("Invalid project namespace stored to config mapping: %s", namespace)
			continue
		}
		projects = append(projects, projectWithOrg{
			Name:           ns.Project,
			Organization:   ns.Owner,
			LocalCheckouts: checkouts,
		})
	}
	sort.SliceStable(projects, func(i, j int) bool {
		return projects[i].Organization < projects[j].Organization
	})

	if len(projects) == 0 {
		r.out.Print(locale.T("project_checkout_empty"))
	} else {
		r.out.Print(projects)
	}

	return nil
}
