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
	orgs, err := model.FetchOrganizations()
	if err != nil {
		return locale.WrapError(err, "organizations_err")
	}

	if len(orgs) == 0 {
		out.Notice(locale.T("organization_no_orgs"))
		return nil
	}

	data, err := newOrgData(orgs)
	if err != nil {
		return locale.WrapError(err, "err_run_orgs_data", "Could not collect information about your organizations.")
	}

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

	tiers, err := model.FetchTiers()
	if err != nil {
		return nil, err
	}

	type tierInfo struct {
		private bool
		title   string
	}
	tiersLookup := make(map[string]tierInfo)
	for _, t := range tiers {
		tiersLookup[t.Name] = tierInfo{
			private: t.RequiresPayment,
			title:   t.Description,
		}
	}

	orgDatas := make([]orgData, len(orgs))
	for i, org := range orgs {
		var tierPrivate bool
		tierTitle := "Unknown"
		t, ok := tiersLookup[org.Tier]
		if ok {
			tierPrivate = t.private
			tierTitle = t.title
		}
		orgDatas[i] = orgData{
			Name:            org.DisplayName,
			URLName:         org.URLname,
			Tier:            tierTitle,
			PrivateProjects: tierPrivate,
		}
	}
	return orgDatas, nil
}
