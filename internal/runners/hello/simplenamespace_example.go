package hello

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
)

// Namespace represents a project namespace of the form <org/project>
type SimpleNamespace struct {
	Owner   string
	Project string
	isSet   bool
}

// NewSimpleNamespace constructs a new instance of SimpleNamespace with the
// provided values. Use a struct literal to construct an empty instance.
func NewSimpleNamespace(owner, project string) *SimpleNamespace {
	return &SimpleNamespace{
		Owner:   owner,
		Project: project,
		isSet:   true,
	}
}

// Set implements the captain flagmarshaler interface.
func (ns *SimpleNamespace) Set(v string) error {
	if ns == nil {
		return errs.New("cannot set nil value")
	}

	partCount := 2
	parts := strings.SplitN(v, "/", partCount)
	if len(parts) < partCount {
		return errs.New("value missing separator '/' (e.g. 'org/project')")
	}

	ns.Owner, ns.Project = parts[0], parts[1]
	ns.isSet = true

	return nil
}

// String implements the fmt.Stringer and flagmarshaler interfaces.
func (ns *SimpleNamespace) String() string {
	if ns == nil {
		return ""
	}

	return fmt.Sprintf("%s/%s", ns.Owner, ns.Project)
}

// Type implements the flagmarshaler interface.
func (ns *SimpleNamespace) Type() string {
	return "simple-namespace"
}

func (ns *SimpleNamespace) IsSet() bool {
	return ns.isSet
}
