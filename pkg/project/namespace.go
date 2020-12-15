package project

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

// NamespaceRegex matches the org and project name in a namespace, eg. ORG/PROJECT
const NamespaceRegex = `^([\w-_]+)\/([\w-_\.]+)(?:#([-a-fA-F0-9]*))?$`

// Namespaced represents a project namespace of the form <OWNER>/<PROJECT>
type Namespaced struct {
	Owner    string
	Project  string
	CommitID *strfmt.UUID
}

type ConfigAble interface {
	Set(key string, value interface{})
	GetString(key string) string
	GetStringSlice(key string) []string
}

func NewNamespace(owner, project, commitID string) *Namespaced {
	ns := &Namespaced{
		owner,
		project,
		nil,
	}
	if commitID != "" {
		commitUUID := strfmt.UUID(commitID)
		ns.CommitID = &commitUUID
	}
	return ns
}

// Set implements the captain argmarshaler interface.
func (ns *Namespaced) Set(v string) error {
	if ns == nil {
		return fmt.Errorf("cannot set nil value")
	}

	parsedNs, err := ParseNamespace(v)
	if err != nil {
		return err
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
func (ns *Namespaced) Validate() error {
	if ns == nil || !ns.IsValid() {
		return locale.NewInputError("err_invalid_namespace", "", ns.String())
	}
	return nil
}

// ParseNamespace returns a valid project namespace
func ParseNamespace(raw string) (*Namespaced, error) {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(raw)
	if len(groups) < 3 {
		return nil, locale.NewInputError("err_invalid_namespace", "", raw)
	}

	names := Namespaced{
		Owner:   groups[1],
		Project: groups[2],
	}

	if len(groups) > 3 && len(groups[3]) > 0 {
		uuid := strfmt.UUID(groups[3])
		names.CommitID = &uuid
	}

	return &names, nil
}

// NameSpaceForConfig returns a valid project namespace.
// This version prefers to create a namespace from a configFile if it exists
func NameSpaceForConfig(configFile string) *Namespaced {
	if !fileutils.FileExists(configFile) {
		return nil
	}

	prj, err := FromPath(configFile)
	if err != nil {
		return nil
	}

	names := Namespaced{
		Owner:   prj.Owner(),
		Project: prj.Name(),
	}

	prjCommitID := prj.CommitID()
	if prjCommitID != "" {
		uuid := strfmt.UUID(prjCommitID)
		names.CommitID = &uuid
	}

	return &names
}

// AvailableProjectPaths returns the paths of all projects associated with the namespace
func AvailableProjectPaths(c ConfigAble, namespace string) []string {
	key := fmt.Sprintf("project_%s", namespace)
	paths := c.GetStringSlice(key)
	paths = funk.FilterString(paths, func(path string) bool {
		return fileutils.FileExists(filepath.Join(path, constants.ConfigFileName))
	})
	paths = funk.UniqString(paths)
	c.Set(key, paths)
	return paths
}
