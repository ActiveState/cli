package organizations

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type Organizations struct {
	out  output.Outputer
	auth *authentication.Auth
}

type primeable interface {
	primer.Outputer
	primer.Auther
}

func NewOrganizations(prime primeable) *Organizations {
	return &Organizations{prime.Output(), prime.Auth()}
}

type OrgParams struct {
}

type orgOutput struct {
	Name            string `json:"name,omitempty"`
	URLName         string `json:"URLName,omitempty" opts:"hidePlain"`
	Tier            string `json:"tier,omitempty" locale:"tier,Tier"`
	PrivateProjects bool   `json:"privateProjects" locale:"privateprojects,Private Projects"`
}

func (o *Organizations) Run(params *OrgParams) error {
	modelOrgs, err := model.FetchOrganizations(o.auth)
	if err != nil {
		return locale.WrapError(err, "organizations_err")
	}

	tiers, err := model.FetchTiers(o.auth)
	if err != nil {
		return err
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

	orgs := make([]orgOutput, len(modelOrgs))
	for i, org := range modelOrgs {
		var tierPrivate bool
		tierTitle := "Unknown"
		t, ok := tiersLookup[org.Tier]
		if ok {
			tierPrivate = t.private
			tierTitle = t.title
		}
		orgs[i] = orgOutput{
			Name:            org.DisplayName,
			URLName:         org.URLname,
			Tier:            tierTitle,
			PrivateProjects: tierPrivate,
		}
	}

	var plainOutput interface{} = orgs
	if len(orgs) == 0 {
		plainOutput = locale.T("organization_no_orgs")
	}
	o.out.Print(output.Prepare(plainOutput, orgs))
	return nil
}
