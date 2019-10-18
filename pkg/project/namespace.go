package project

import (
	"regexp"

	"github.com/ActiveState/cli/internal/failures"
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
