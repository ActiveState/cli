package project

import (
	"fmt"
	"regexp"

	"github.com/ActiveState/cli/internal/locale"
)

// NamespaceRegex matches the org and project name in a namespace, eg. ORG/PROJECT
const NamespaceRegex = `^([\w-_]+)\/([\w-_\.]+)$`

// Namespace represents a project namespace of the form <OWNER>/<PROJECT>
type Namespace struct {
	Owner   string
	Project string
}

func NewNamespace(owner, project string) *Namespace {
	ns := &Namespace{
		owner,
		project,
	}
	return ns
}

// Set implements the captain argmarshaler interface.
func (ns *Namespace) Set(v string) error {
	if ns == nil {
		return fmt.Errorf("cannot set nil value")
	}

	parsedNs, err := parseNamespace(v)
	if err != nil {
		return err
	}

	*ns = *parsedNs
	return nil
}

// String implements the fmt.Stringer interface.
func (ns *Namespace) String() string {
	if ns == nil {
		return ""
	}

	var sep string
	if ns.IsValid() {
		sep = "/"
	}
	return fmt.Sprintf("%s%s%s", ns.Owner, sep, ns.Project)
}

// Type returns the human readable type name of Namespace.
func (ns *Namespace) Type() string {
	return "namespace"
}

// IsValid returns whether or not the namespace is set sufficiently.
func (ns *Namespace) IsValid() bool {
	return ns != nil && ns.Owner != "" && ns.Project != ""
}

// Validate returns a failure if the namespace is not valid.
func (ns *Namespace) Validate() error {
	if ns == nil || !ns.IsValid() {
		return locale.NewInputError("err_invalid_namespace", "", ns.String())
	}
	return nil
}

// parseNamespace returns a valid project namespace
func parseNamespace(raw string) (*Namespace, error) {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(raw)
	if len(groups) < 3 {
		return nil, locale.NewInputError("err_invalid_namespace", "", raw)
	}

	names := Namespace{
		Owner:   groups[1],
		Project: groups[2],
	}

	return &names, nil
}

