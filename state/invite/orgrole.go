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

// orgRoleChoices returns a localized description of organization roles, and a mapping of these strings to their org role
//
// The values can be used in a `prompter.Select()` call:
//  - The string array as the localized and sorted list of `choices`
//  - The map to interpret the results back to an organization role
func orgRoleChoices() ([]string, map[string]OrgRole) {
	return []string{
			locale.T("org_role_choice_owner"),
			locale.T("org_role_choice_member"),
		}, map[string]OrgRole{
			locale.T("org_role_choice_owner"):  Owner,
			locale.T("org_role_choice_member"): Member,
		}
}
