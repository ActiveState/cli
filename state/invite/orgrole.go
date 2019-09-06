package invite

import "github.com/ActiveState/cli/internal/locale"

// OrgRole is an enumeration of the roles that user can have in an organization
type OrgRole int

const (
	// None means no role selected
	None OrgRole = iota
	// Owner of an organization
	Owner
	// Member in an organization
	Member
)

// orgRoleChoices returns the valid choices for organization member roles
func orgRoleChoices() []string {
	return []string{
		locale.T("org_role_choice_owner"),
		locale.T("org_role_choice_member"),
	}
}
