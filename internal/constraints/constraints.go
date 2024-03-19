package constraints

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/thoas/go-funk"
)

var cache = make(map[string]interface{})

func getCache(key string, getter func() (interface{}, error)) (interface{}, error) {
	if v, ok := cache[key]; ok {
		return v, nil
	}
	v, err := getter()
	if err != nil {
		return nil, err
	}
	cache[key] = v
	return v, err
}

// For testing.
var osOverride, osVersionOverride, archOverride, libcOverride, compilerOverride string

type Conditional struct {
	params map[string]interface{}
	funcs  template.FuncMap
}

func NewConditional(a *authentication.Auth) *Conditional {
	c := &Conditional{map[string]interface{}{}, map[string]interface{}{}}

	c.RegisterFunc("Mixin", func() map[string]interface{} {
		res := map[string]string{
			"Name":  "",
			"Email": "",
		}
		if a.Authenticated() {
			res["Name"] = a.WhoAmI()
			res["Email"] = a.Email()
		}
		return map[string]interface{}{
			"User": res,
		}
	})
	c.RegisterFunc("Contains", funk.Contains)
	c.RegisterFunc("HasPrefix", strings.HasPrefix)
	c.RegisterFunc("HasSuffix", strings.HasSuffix)
	c.RegisterFunc("MatchRx", func(rxv, v string) bool {
		rx, err := regexp.Compile(rxv)
		if err != nil {
			logging.Warning("Invalid Regex: %s, error: %v", rxv, err)
			return false
		}
		return rx.Match([]byte(v))
	})

	return c
}

type projectable interface {
	Source() *projectfile.Project
	Owner() string
	Name() string
	NamespaceString() string
	BranchName() string
	Path() string
	Dir() string
	URL() string
	LegacyCommitID() string
	SetLegacyCommit(string) error
}

func NewPrimeConditional(auth *authentication.Auth, pj projectable, subshellName string) *Conditional {
	var (
		pjOwner     string
		pjName      string
		pjNamespace string
		pjURL       string
		pjCommit    string
		pjBranch    string
		pjDir       string
	)
	if !ptr.IsNil(pj) {
		pjOwner = pj.Owner()
		pjName = pj.Name()
		pjNamespace = pj.NamespaceString()
		pjURL = pj.URL()
		pjCommit = pj.LegacyCommitID() // Not using localcommit due to import cycle. See anti-pattern comment in localcommit pkg.
		pjBranch = pj.BranchName()
		pjDir = pj.Dir()
	}

	c := NewConditional(auth)
	c.RegisterParam("Project", map[string]string{
		"Owner":     pjOwner,
		"Name":      pjName,
		"Namespace": pjNamespace,
		"Url":       pjURL,
		"Commit":    pjCommit,
		"Branch":    pjBranch,
		"Path":      pjDir,

		// Legacy
		"NamespacePrefix": pjNamespace,
	})
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		multilog.Error("Could not detect OSVersion: %v", err)
	}
	c.RegisterParam("OS", map[string]interface{}{
		"Name":         sysinfo.OS().String(),
		"Version":      osVersion,
		"Architecture": sysinfo.Architecture().String(),
	})
	c.RegisterParam("Shell", subshellName)

	return c
}

func (c *Conditional) RegisterFunc(name string, value interface{}) {
	c.funcs[name] = value
}

func (c *Conditional) RegisterParam(name string, value interface{}) {
	c.params[name] = value
}

func (c *Conditional) Eval(conditional string) (bool, error) {
	tpl, err := template.New("letter").Funcs(c.funcs).Parse(fmt.Sprintf(`{{if %s}}1{{end}}`, conditional))
	if err != nil {
		return false, locale.WrapInputError(err, "err_conditional", "Invalid 'if' condition: '{{.V0}}', error: '{{.V1}}'.", conditional, err.Error())
	}

	result := bytes.Buffer{}
	if err := tpl.Execute(&result, c.params); err != nil {
		return false, locale.WrapInputError(err, "err_conditional", "Invalid 'if' condition: '{{.V0}}', error: '{{.V1}}'.", conditional, err.Error())
	}

	return result.String() == "1", nil
}

// FilterUnconstrained filters a list of constrained entities and returns only
// those which are unconstrained. If two items with the same name exist, only
// the most specific item will be added to the results.
func FilterUnconstrained(conditional *Conditional, items []projectfile.ConstrainedEntity) ([]projectfile.ConstrainedEntity, error) {
	type itemIndex struct {
		specificity int
		index       int
	}
	selected := make(map[string]itemIndex)

	if conditional == nil {
		multilog.Error("FilterUnconstrained called with nil conditional")
	}

	for i, item := range items {
		if conditional != nil && item.ConditionalFilter() != "" {
			isTrue, err := conditional.Eval(string(item.ConditionalFilter()))
			if err != nil {
				return nil, err
			}

			if isTrue {
				selected[item.ID()] = itemIndex{0, i}
			}
		}

		if item.ConditionalFilter() == "" {
			selected[item.ID()] = itemIndex{0, i}
		}
	}
	indices := make([]int, 0, len(selected))
	for _, s := range selected {
		indices = append(indices, s.index)
	}
	// ensure that the items are returned in the order we get them
	sort.Ints(indices)
	var res []projectfile.ConstrainedEntity
	for _, index := range indices {
		res = append(res, items[index])
	}
	return res, nil
}
