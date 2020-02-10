package project

import (
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

// Namespaced represents a project namespace of the form <OWNER>/<PROJECT>
type Namespaced struct {
	Owner   string
	Project string
}

// ParseNamespace returns a valid project namespace
func ParseNamespace(raw string) (*Namespaced, *failures.Failure) {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(raw)
	if len(groups) != 3 {
		return nil, FailInvalidNamespace.New(locale.Tr("err_invalid_namespace", raw))
	}

	return &Namespaced{
		Owner:   groups[1],
		Project: groups[2],
	}, nil
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
		return &names, nil
	}

	return ParseNamespace(raw)
}
