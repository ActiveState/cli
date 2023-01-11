package projects

import (
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Holds a union of project and organization parameters.
type projectWithOrg struct {
	Name           string   `json:"name"`
	Organization   string   `json:"organization"`
	LocalCheckouts []string `json:"local_checkouts,omitempty"`
	Executables    []string `json:"executables,omitempty"`
}

type projectWithOrgs []projectWithOrg

func newProjectWithOrg(name, org string, checkouts []string) *projectWithOrg {
	p := projectWithOrg{Name: name, Organization: org, LocalCheckouts: checkouts}
	for _, checkout := range checkouts {
		var execDir string
		if proj, err := project.FromPath(checkout); err == nil {
			projectTarget := target.NewProjectTarget(proj, storage.CachePath(), nil, "")
			execDir = setup.ExecDir(projectTarget.Dir())
		} else {
			multilog.Error("Unable to get project %s from checkout: %v", checkout, err)
		}
		p.Executables = append(p.Executables, execDir)
	}
	return &p
}

func (o projectWithOrgs) MarshalOutput(f output.Format) interface{} {
	type projectOutputPlain struct {
		Name           string
		Organization   string
		LocalCheckouts string `locale:"local_checkouts,Local Checkouts" opts:"emptyNil,separateLine"`
		Executables    string `locale:"executables,Executables" opts:"emptyNil,separateLine"`
	}

	if f != output.PlainFormatName {
		return o
	}

	r := []projectOutputPlain{}
	for _, v := range o {
		checkouts := []string{}
		executables := []string{}
		for i, checkout := range v.LocalCheckouts {
			execDir := v.Executables[i]
			if execDir != "" {
				checkouts = append(checkouts, locale.Tl("projects_local_checkout_exec", " ├─ Local Checkout → {{.V0}}", checkout))
				if f == output.PlainFormatName {
					// Show executable path below checkout path for plain text output.
					checkouts = append(checkouts, locale.Tl("projects_executables", " └─ Executables → {{.V0}}", execDir))
				} else {
					// Show executables in a separate table.
					executables = append(executables, execDir)
				}
			} else {
				checkouts = append(checkouts, locale.Tl("projects_local_checkout", " └─ Local Checkout → {{.V0}}", checkout))
			}
		}
		r = append(r, projectOutputPlain{v.Name, v.Organization, strings.Join(checkouts, "\n"), strings.Join(executables, "\n")})
	}
	return r
}

type configGetter interface {
	projectfile.ConfigGetter
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
	primer.Configurer
}

func NewParams() *Params {
	return &Params{Local: false}
}

func NewProjects(prime primeable) *Projects {
	return newProjects(prime.Auth(), prime.Output(), prime.Config())
}

func newProjects(auth *authentication.Auth, out output.Outputer, config configGetter) *Projects {
	return &Projects{
		auth,
		out,
		config,
	}
}

func (r *Projects) Run(params *Params) error {
	localProjects := projectfile.GetProjectMapping(r.config)

	var projects projectWithOrgs = []projectWithOrg{}
	for namespace, checkouts := range localProjects {
		ns, err := project.ParseNamespace(namespace)
		if err != nil {
			multilog.Error("Invalid project namespace stored to config mapping: %s", namespace)
			continue
		}

		projects = append(projects, *newProjectWithOrg(ns.Project, ns.Owner, checkouts))
	}
	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].Organization == projects[j].Organization {
			return projects[i].Name < projects[j].Name
		}
		return projects[i].Organization < projects[j].Organization
	})

	if len(projects) == 0 {
		r.out.Print(locale.T("project_checkout_empty"))
	} else {
		r.out.Print(projects)
	}

	return nil
}
