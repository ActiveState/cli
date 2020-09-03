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
const NamespaceRegex = `^([\w-_]+)\/([\w-_\.]+)(?:#([-a-fA-F0-9]*))?$`

// Namespaced represents a project namespace of the form <OWNER>/<PROJECT>
type Namespaced struct {
	Owner    string
	Project  string
	CommitID string
}

// Set implements the captain argmarshaler interface.
func (ns *Namespaced) Set(v string) error {
	if ns == nil {
		return fmt.Errorf("cannot set nil value")
	}

	parsedNs, fail := ParseNamespace(v)
	if fail != nil {
		return fail
	}

	*ns = *parsedNs
	return nil
}

// String implements the fmt.Stringer interface.
func (ns *Namespaced) String() string {
	if ns == nil {
		return ""
	}

	var sep string
	if ns.IsValid() {
		sep = "/"
	}
	return fmt.Sprintf("%s%s%s", ns.Owner, sep, ns.Project)
}

// Type returns the human readable type name of Namespaced.
func (ns *Namespaced) Type() string {
	return "namespace"
}

// IsValid returns whether or not the namespace is set sufficiently.
func (ns *Namespaced) IsValid() bool {
	return ns != nil && ns.Owner != "" && ns.Project != ""
}

// Validate returns a failure if the namespace is not valid.
func (ns *Namespaced) Validate() *failures.Failure {
	if ns == nil || !ns.IsValid() {
		return FailInvalidNamespace.New(locale.Tr("err_invalid_namespace", ns.String()))
	}
	return nil
}

// ParseNamespace returns a valid project namespace
func ParseNamespace(raw string) (*Namespaced, *failures.Failure) {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(raw)
	if len(groups) < 3 {
		return nil, FailInvalidNamespace.New(locale.Tr("err_invalid_namespace", raw))
	}

	names := Namespaced{
		Owner:   groups[1],
		Project: groups[2],
	}

	if len(groups) > 3 {
		names.CommitID = groups[3]
	}

	return &names, nil
}

// ParseNamespaceOrConfigfile returns a valid project namespace.
// This version prefers to create a namespace from a configFile if it exists
func ParseNamespaceOrConfigfile(raw string, configFile string) (*Namespaced, *failures.Failure) {

	if fileutils.FileExists(configFile) {
		prj, fail := FromPath(configFile)
		if fail != nil {
			return nil, FailInputSecretValue.New(locale.Tr("err_invalid_namespace", raw))
		}
		var names Namespaced
		names.Owner = prj.Owner()
		names.Project = prj.Name()
		names.CommitID = prj.CommitID()
		return &names, nil
	}

	return ParseNamespace(raw)
}
