package projects

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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
	LastUsed       []string `json:"last_used,omitempty"`
}

const runtimeInUseCutoff = 90 * time.Second // heartbeats are usually sent once per minute

func newProjectWithOrg(name, org string, checkouts []string, lastUsedTimes map[string]interface{}) projectWithOrg {
	p := projectWithOrg{Name: name, Organization: org, LocalCheckouts: checkouts}
	for _, checkout := range checkouts {
		var execDir string
		if proj, err := project.FromPath(checkout); err == nil {
			projectTarget := target.NewProjectTarget(proj, nil, "")
			execDir = setup.ExecDir(projectTarget.Dir())
		} else {
			multilog.Error("Unable to get project %s from checkout: %v", checkout, err)
		}
		p.Executables = append(p.Executables, execDir)
		lastUsed := locale.Tl("projects_last_use_unknown", "unknown")
		if v, exists := lastUsedTimes[execDir]; exists {
			if t, err := time.Parse(time.RFC3339, v.(string)); err == nil {
				if time.Since(t) <= runtimeInUseCutoff && os.Getenv(constants.RuntimeInUseNoCutoffTimeEnvVarName) == "" {
					lastUsed = locale.Tl("projects_last_use_currently_in_use", "currently in use")
				} else {
					lastUsed = t.Format(time.DateTime)
				}
			} else {
				multilog.Error("Last used time was not a datetime string")
			}
		}
		p.LastUsed = append(p.LastUsed, lastUsed)
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
		LastUsed       string `locale:"last_used,Last Used" opts:"emptyNil,separateLine"`
	}

	r := []projectOutputPlain{}
	for _, v := range o.projects {
		checkouts := []string{}
		executables := []string{}
		lastUseds := []string{}
		for i, checkout := range v.LocalCheckouts {
			if len(v.Executables) <= i {
				continue // remote project listings do not have executables
			}
			execDir := v.Executables[i]
			lastUsed := v.LastUsed[i]
			if execDir != "" || lastUsed != "" {
				checkouts = append(checkouts, locale.Tl("projects_local_checkout_exec", " ├─ Local Checkout → {{.V0}}", checkout))
				if f == output.PlainFormatName {
					// Show executable path and last used below checkout path for plain text output.
					checkouts = append(checkouts, locale.Tl("projects_executables", " ├─ Executables → {{.V0}}", execDir))
					checkouts = append(checkouts, locale.Tl("projects_lastused", " └─ Last Used → {{.V0}}", lastUsed))
				} else {
					// Show executables and last used in a separate table.
					executables = append(executables, execDir)
					lastUseds = append(lastUseds, lastUsed)
				}
			} else {
				checkouts = append(checkouts, locale.Tl("projects_local_checkout", " └─ Local Checkout → {{.V0}}", checkout))
			}
		}
		r = append(r, projectOutputPlain{v.Name, v.Organization, strings.Join(checkouts, "\n"), strings.Join(executables, "\n"), strings.Join(lastUseds, "\n")})
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
	auth     *authentication.Auth
	out      output.Outputer
	config   configGetter
	svcModel *model.SvcModel
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Configurer
	primer.Prompter
	primer.SvcModeler
}

func NewParams() *Params {
	return &Params{Local: false}
}

func NewProjects(prime primeable) *Projects {
	return newProjects(prime.Auth(), prime.Output(), prime.Config(), prime.SvcModel())
}

func newProjects(auth *authentication.Auth, out output.Outputer, config configGetter, svcModel *model.SvcModel) *Projects {
	return &Projects{
		auth,
		out,
		config,
		svcModel,
	}
}

func (r *Projects) Run(params *Params) error {
	localProjects := projectfile.GetProjectMapping(r.config)
	var projects []projectWithOrg

	lastUsed, err := r.svcModel.CheckRuntimeLastUsed(context.Background())
	if err != nil {
		multilog.Error("Unable to determine runtime last used times: %v", err)
	}

	for namespace, checkouts := range localProjects {
		ns, err := project.ParseNamespace(namespace)
		if err != nil {
			multilog.Error("Invalid project namespace stored to config mapping: %s", namespace)
			continue
		}
		projects = append(projects, newProjectWithOrg(ns.Project, ns.Owner, checkouts, lastUsed.Times))
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
