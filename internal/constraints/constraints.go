package constraints

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/thoas/go-funk"
)

// For testing.
var osOverride, osVersionOverride, archOverride, libcOverride, compilerOverride string

type Conditional struct {
	params map[string]interface{}
	funcs  template.FuncMap
}

func NewConditional() *Conditional {
	c := &Conditional{map[string]interface{}{}, map[string]interface{}{}}

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

func NewPrimeConditional(val interface{}) *Conditional {
	c := NewConditional()

	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	to := v.Type()
	fields := reflect.VisibleFields(to)

	for _, f := range fields {
		sv := v.FieldByIndex(f.Index)
		if sv.Kind() == reflect.Ptr {
			sv = sv.Elem()
		}
		sto := sv.Type()

		switch sto.Kind() {
		case reflect.Func:
			c.RegisterFunc(f.Name, sv.Interface())

		default:
			c.RegisterParam(f.Name, sv.Interface())
		}
	}

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
	tpl.Execute(&result, c.params)

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
