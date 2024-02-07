package project

import (
	"fmt"
	"regexp"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
)

// NamespaceRegex matches the org and project name in a namespace, eg. org/project
const NamespaceRegex = `^([\w-_]+)\/([\w-_\.]+)(?:#([-a-fA-F0-9]*))?$`

// ProjectRegex matches the project name for a namespace with omitted org.
const ProjectRegex = `^([\w-_\.]+)(?:#([-a-fA-F0-9]*))?$`

// Namespaced represents a project namespace of the form <org/project>
type Namespaced struct {
	Owner          string
	Project        string
	CommitID       *strfmt.UUID
	AllowOmitOwner bool
}

type ConfigAble interface {
	projectfile.ConfigGetter
}

func NewNamespace(owner, project, commitID string) *Namespaced {
	ns := &Namespaced{
		Owner:   owner,
		Project: project,
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
		if !ns.AllowOmitOwner {
			return err
		}
		parsedNs, err = ParseProjectNoOwner(v)
		if err != nil {
			return err
		}
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
	return ns != nil && ns.Project != ""
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
		uuidString := groups[3]
		if !strfmt.IsUUID(uuidString) {
			return nil, locale.NewInputError("err_invalid_commit_id", "", uuidString)
		}
		uuid := strfmt.UUID(uuidString)
		names.CommitID = &uuid
	}

	return &names, nil
}

func ParseProjectNoOwner(raw string) (*Namespaced, error) {
	rx := regexp.MustCompile(ProjectRegex)
	groups := rx.FindStringSubmatch(raw)
	if len(groups) < 2 {
		return nil, locale.NewInputError("err_invalid_project_name", "", raw)
	}

	names := Namespaced{
		Project:        groups[1],
		AllowOmitOwner: true,
	}

	if len(groups) > 2 && len(groups[2]) > 0 {
		uuidString := groups[2]
		if !strfmt.IsUUID(uuidString) {
			return nil, locale.NewInputError("err_invalid_commit_id", "", uuidString)
		}
		uuid := strfmt.UUID(uuidString)
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

	commitID := strfmt.UUID(prj.LegacyCommitID()) // Not using localcommit due to import cycle. See anti-pattern comment in localcommit pkg.
	if commitID != "" {
		names.CommitID = &commitID
	}

	return &names
}
