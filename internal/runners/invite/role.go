package invite

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Role is an enumeration of the roles that user can have in an organization
type Role int

const (
	Unknown Role = iota
	// Owner of an organization
	Owner
	// Member in an organization
	Member
)

func roleNames() []string {
	return []string{Owner.String(), Member.String()}
}

func (r Role) Type() string {
	return "Role"
}

func (r Role) String() string {
	switch r {
	case Owner:
		return "owner"
	case Member:
		return "member"
	default:
		return locale.Tl("unknown", "Unknown")
	}
}

func (r *Role) Set(v string) error {
	switch v {
	case "owner":
		*r = Owner
	case "member":
		*r = Member
	case "":
		*r = Member
	default:
		*r = Unknown
		return locale.NewInputError("err_invite_invalid_role", "Invalid role: {{.V0}}, should be one of: {{.V1}}", v, strings.Join(roleNames(), ", "))
	}
	return nil
}
