package project

import (
	"fmt"
	"regexp"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

// FailInvalidNamespace indicates the provided string is not a valid
// representation of a project namespace
var FailInvalidNamespace = failures.Type("project.fail.invalidnamespace", failures.FailUserInput)

// NamespaceRegex matches the org and project name in a namespace, eg. ORG/PROJECT
const NamespaceRegex = `^([\w-_]+)\/([\w-_\.]+)$`

// Namespace represents a project namespace of the form <OWNER>/<PROJECT>
type Namespace struct {
	Owner   string
	Project string
}

// Set implements the captain argmarshaler interface.
func (ns *Namespace) Set(v string) error {
	if ns == nil {
		return fmt.Errorf("cannot set nil value")
	}

	nsx, fail := ParseNamespace(v)
	if fail != nil {
		return fail
	}

	*ns = *nsx
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

// IsValid returns whether or not the namespace is set sufficiently.
func (ns *Namespace) IsValid() bool {
	return ns != nil && ns.Owner != "" && ns.Project != ""
}

// ParseNamespace returns a valid project namespace
func ParseNamespace(raw string) (*Namespace, *failures.Failure) {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(raw)
	if len(groups) != 3 {
		return nil, FailInvalidNamespace.New(locale.Tr("err_invalid_namespace", raw))
	}

	return &Namespace{
		Owner:   groups[1],
		Project: groups[2],
	}, nil
}

// ParseNamespaceOrConfigfile returns a valid project namespace.
// This version prefers to create a namespace from a configFile if it exists
func ParseNamespaceOrConfigfile(raw string, configFile string) (*Namespace, *failures.Failure) {

	if fileutils.FileExists(configFile) {
		prj, fail := FromPath(configFile)
		if fail != nil {
			return nil, FailInputSecretValue.New(locale.Tr("err_invalid_namespace", raw))
		}
		var names Namespace
		names.Owner = prj.Owner()
		names.Project = prj.Name()
		return &names, nil
	}

	return ParseNamespace(raw)
}
