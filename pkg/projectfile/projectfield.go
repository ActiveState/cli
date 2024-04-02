package projectfile

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
)

var projectFieldRE = regexp.MustCompile(`(?m:^project:["' ]*(https?:\/\/.*?)["' ]*$)`)

type projectField struct {
	url *url.URL
}

func NewProjectField() *projectField {
	return &projectField{}
}

func (p *projectField) LoadProject(rawProjectValue string) error {
	pv := rawProjectValue
	u, err := url.Parse(pv)
	if err != nil {
		return locale.NewInputError("err_project_url", "Invalid format for project field: {{.V0}}.", pv)
	}
	p.url = u

	return nil
}

func (p *projectField) String() string {
	return p.url.String()
}

func (p *projectField) SetNamespace(owner, name string) {
	p.setPath(fmt.Sprintf("%s/%s", owner, name))
}

func (p *projectField) SetBranch(branch string) {
	p.setQuery("branch", branch)
}

func (p *projectField) SetLegacyCommitID(commitID string) {
	p.setQuery("commitID", commitID)
}

func (p *projectField) setPath(path string) {
	p.url.Path = path
	p.url.RawPath = p.url.EscapedPath()
}

func (p *projectField) setQuery(key, value string) {
	q := p.url.Query()
	q.Set(key, value)
	p.url.RawQuery = q.Encode()
}

func (p *projectField) Marshal() string {
	return p.url.String()
}

func (p *projectField) Save(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errs.Wrap(err, "os.ReadFile %s failed", path)
	}

	projectValue := p.url.String()
	out := projectFieldRE.ReplaceAll(data, []byte("project: "+projectValue))
	if !strings.Contains(string(out), projectValue) {
		return locale.NewInputError("err_set_project")
	}

	if err := os.WriteFile(path, out, 0664); err != nil {
		return errs.Wrap(err, "os.WriteFile %s failed", path)
	}

	return nil
}
