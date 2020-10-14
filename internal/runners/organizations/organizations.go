package organizations

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Organizations struct {
	output.Outputer
}

type primeable interface {
	primer.Outputer
}

func NewOrganizations(prime primeable) *Organizations {
	return &Organizations{prime.Output()}
}

type OrgParams struct {
}

// Run the organizations command.
func (o *Organizations) Run(params *OrgParams) error {
	return run(params, o.Outputer)
}

func run(params *OrgParams, out output.Outputer) error {
	orgs, fail := model.FetchOrganizations()
	if fail != nil {
		return fail.WithDescription("organizations_err").ToError()
	}

	if len(orgs) == 0 {
		out.Notice(locale.T("organization_no_orgs"))
		return nil
	}

	data, err := newOrgData(orgs)
	if err != nil {
		return locale.WrapError(err, "err_run_orgs_data", "Could not collect information about your organizations.")
	}

	out.Print(locale.Tl("organizations_list_info", "Here are the organizations you are a part of."))
	out.Print(data)
	return nil
}

type orgData struct {
	Name            string `json:"name,omitempty"`
	URLName         string `json:"URLName,omitempty" opts:"hidePlain"`
	Tier            string `json:"tier,omitempty" locale:"tier,Tier"`
	PrivateProjects bool   `json:"privateProjects" locale:"privateprojects,Private Projects"`
}

func newOrgData(orgs []*mono_models.Organization) ([]orgData, error) {

	tiers, fail := model.FetchTiers()
	if fail != nil {
		return nil, fail.ToError()
	}
	tiersToPrivMap := make(map[string]bool)
	for _, t := range tiers {
		tiersToPrivMap[t.Name] = t.RequiresPayment
	}

	orgDatas := make([]orgData, len(orgs))
	for i, org := range orgs {
		priv, _ := tiersToPrivMap[org.Tier]
		orgDatas[i] = orgData{
			Name:            org.Name,
			URLName:         org.URLname,
			Tier:            org.Tier,
			PrivateProjects: priv,
		}
	}
	return orgDatas, nil
}
