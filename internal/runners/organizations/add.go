package organizations

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type OrganizationsAdd struct {
	out output.Outputer
}

func NewOrganizationsAdd(prime primeable) *OrganizationsAdd {
	return &OrganizationsAdd{prime.Output()}
}

type OrgAddParams struct {
	Name string
}

func (o *OrganizationsAdd) Run(params *OrgAddParams) error {
	if err := model.CreateOrg(params.Name); err != nil {
		return locale.WrapError(err, "err_organizations_add", "Could not create organization")
	}
	o.out.Notice(locale.Tl("organizations_add_success", "Organization created"))
	return nil
}
