package invite

import (
	"strconv"

	"github.com/ActiveState/cli/internal-as/locale"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Org struct {
	*mono_models.Organization
}

func (o *Org) Type() string {
	return "Org"
}

func (o *Org) String() string {
	if o == nil || o.Organization == nil {
		return ""
	}
	return o.Organization.URLname
}

func (o *Org) Set(v string) error {
	var err error
	o.Organization, err = model.FetchOrgByURLName(v)
	return err
}

func (o *Org) CanInvite(numInvites int) error {
	// don't allow personal organizations
	if o.Personal {
		return locale.NewInputError("err_invite_personal", "This project does not belong to any organization and so cannot have any users invited to it. To invite users create an organization.")
	}

	limits, err := model.FetchOrganizationLimits(o.URLname)
	if err != nil {
		return locale.WrapError(err, "err_invite_fetchlimits", "Could not detect member limits for organization.")
	}

	requestedMemberCount := o.MemberCount + int64(numInvites)
	if limits.UsersLimit > 0 && requestedMemberCount > limits.UsersLimit { // UsersLimit=0 means unlimited
		return locale.NewInputError("err_invite_limit",
			"Only {{.V0}} users can be added to the organization '{{.V1}}'. To add more users please upgrade your organization.",
			strconv.FormatInt(limits.UsersLimit-o.MemberCount, 10), o.String())
	}
	return nil
}
