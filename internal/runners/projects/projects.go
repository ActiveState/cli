package projects

import (
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

// Holds a union of project and organization parameters.
type projectWithOrg struct {
	Name           string   `json:"name"`
	Organization   string   `json:"organization"`
	LocalCheckouts []string `json:"local_checkouts,omitempty"`
	Executables    []string `json:"executables,omitempty"`
}

func newProjectWithOrg(prime primeable, name, org string, checkouts []string) projectWithOrg {
	p := projectWithOrg{Name: name, Organization: org, LocalCheckouts: checkouts}
	for _, checkout := range checkouts {
		var execDir string
		if proj, err := project.FromPath(checkout); err == nil {
			execDir = runtime_helpers.ExecutorPathFromProject(proj)
		} else {
			multilog.Error("Unable to get project %s from checkout: %v", checkout, err)
		}
		p.Executables = append(p.Executables, execDir)
	}
	return p
}

type projectsOutput struct {
	projects []projectWithOrg
}

func (o *projectsOutput) MarshalOutput(f output.Format) interface{} {
	if len(o.projects) == 0 {
		return locale.T("project_checkout_empty")
	}

	type projectOutputPlain struct {
		Name           string
		Organization   string
		LocalCheckouts string `locale:"local_checkouts,Local Checkouts" opts:"emptyNil,separateLine"`
		Executables    string `locale:"executables,Executables" opts:"emptyNil,separateLine"`
	}

	r := []projectOutputPlain{}
	for _, v := range o.projects {
		checkouts := []string{}
		executables := []string{}
		for i, checkout := range v.LocalCheckouts {
			if len(v.Executables) <= i {
				continue // remote project listings do not have executables
			}
			execDir := v.Executables[i]
			if execDir != "" {
				checkouts = append(checkouts, locale.Tl("projects_local_checkout_exec", " {{.V0}} Local Checkout → {{.V1}}", output.TreeMid, checkout))
				if f == output.PlainFormatName {
					// Show executable path below checkout path for plain text output.
					checkouts = append(checkouts, locale.Tl("projects_executables", " {{.V0}} Executables → {{.V1}}", output.TreeEnd, execDir))
				} else {
					// Show executables in a separate table.
					executables = append(executables, execDir)
				}
			} else {
				checkouts = append(checkouts, locale.Tl("projects_local_checkout", " {{.V0}} Local Checkout → {{.V1}}", output.TreeEnd, checkout))
			}
		}
		r = append(r, projectOutputPlain{v.Name, v.Organization, strings.Join(checkouts, "\n"), strings.Join(executables, "\n")})
	}
	return r
}

func (o *projectsOutput) MarshalStructured(f output.Format) interface{} {
	return o.projects
}

type configGetter interface {
	projectfile.ConfigGetter
}

// Params are command line parameters
type Params struct {
	Local bool // Whether to show locally checked out projects only
}

type Projects struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	auth   *authentication.Auth
	out    output.Outputer
	config configGetter
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Configurer
	primer.Prompter
}

func NewParams() *Params {
	return &Params{Local: false}
}

func NewProjects(prime primeable) *Projects {
	return &Projects{
		prime,
		prime.Auth(),
		prime.Output(),
		prime.Config(),
	}
}

func (r *Projects) Run(params *Params) error {
	localProjects := projectfile.GetProjectMapping(r.config)
	var projects []projectWithOrg

	for namespace, checkouts := range localProjects {
		ns, err := project.ParseNamespace(namespace)
		if err != nil {
			multilog.Error("Invalid project namespace stored to config mapping: %s", namespace)
			continue
		}
		projects = append(projects, newProjectWithOrg(r.prime, ns.Project, ns.Owner, checkouts))
	}

	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].Organization == projects[j].Organization {
			return projects[i].Name < projects[j].Name
		}
		return projects[i].Organization < projects[j].Organization
	})

	r.out.Print(&projectsOutput{projects})
	return nil
}
