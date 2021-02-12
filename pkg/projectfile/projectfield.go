package projectfile

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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
	wasHeadless := p.isHeadless()
	pathNodes := p.pathNodes()
	p.setPath(fmt.Sprintf("%s/%s", owner, name))
	if wasHeadless && len(pathNodes) == 2 {
		p.SetCommit(pathNodes[1], false)
	}
}

func (p *projectField) SetCommit(commitID string, headless bool) {
	if headless {
		p.setPath("/commit/" + commitID)
		p.unsetQuery("commitID")
		p.unsetQuery("branch")
	} else {
		p.setQuery("commitID", commitID)
	}
}

func (p *projectField) SetBranch(branch string) {
	if p.isHeadless() {
		logging.Error("Ignoring SetBranch when project is headless")
		return
	}
	p.setQuery("branch", branch)
}

func (p *projectField) pathNodes() []string {
	return strings.Split(strings.Trim(p.url.Path, "/"), "/")
}

func (p *projectField) isHeadless() bool {
	return strings.HasPrefix(p.url.Path, "/commit/")
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

func (p *projectField) unsetQuery(key string) {
	q := p.url.Query()
	q.Del(key)
	p.url.RawQuery = q.Encode()
}

func (p *projectField) Marshal() string {
	return p.url.String()
}

func (p *projectField) Save(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return errs.Wrap(err, "ioutil.ReadFile %s failed", path)
	}

	projectValue := p.url.String()
	out := projectFieldRE.ReplaceAll(data, []byte("project: "+projectValue))
	if !strings.Contains(string(out), projectValue) {
		return locale.NewInputError("err_set_project")
	}

	if err := ioutil.WriteFile(path, out, 0664); err != nil {
		return errs.Wrap(err, "ioutil.WriteFile %s failed", path)
	}

	return nil
}
