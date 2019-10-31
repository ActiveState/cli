package organizations

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Flags captures values for any of the flags used with the organizations command.
var Flags struct {
	JSON bool
}

// Command is the organization command's definition.
var Command = &commands.Command{
	Name:        "organizations",
	Aliases:     []string{"orgs"},
	Description: "organizations_description",
	Run:         Execute,
	Flags: []*commands.Flag{
		{
			Name:        "json",
			Description: "flag_json_desc",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.JSON,
		},
	},
}

// Execute the organizations command.
func Execute(cmd *cobra.Command, args []string) {
	orgs, fail := model.FetchOrganizations()
	if fail != nil {
		failures.Handle(fail, locale.T("organizations_err"))
		return
	}

	if Flags.JSON {
		data, fail := orgsAsJSON(orgs)
		if fail != nil {
			failures.Handle(fail, locale.T("organizations_err_output"))
			return
		}

		print.Line(string(data))
		return
	}

	listOrganizations(orgs)
}

func orgsAsJSON(orgs []*mono_models.Organization) ([]byte, *failures.Failure) {
	type orgRaw struct {
		Name            string `json:"name,omitempty"`
		URLName         string `json:"URLName,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}

	tiers, fail := model.FetchTiers()
	if fail != nil {
		return nil, fail
	}
	tiersToPrivMap := make(map[string]bool)
	for _, t := range tiers {
		tiersToPrivMap[t.Name] = t.RequiresPayment
	}

	orgsRaw := make([]orgRaw, len(orgs))
	for i, org := range orgs {
		if val, ok := tiersToPrivMap[org.Tier]; ok {
			orgsRaw[i] = orgRaw{
				Name:            org.Name,
				URLName:         org.Urlname,
				Tier:            org.Tier,
				PrivateProjects: val,
			}
		} else {
			return nil, failures.FailNotFound.New(locale.T("organizations_unknown_tier", map[string]string{
				"Tier":         org.Tier,
				"Organization": org.Name,
			}))
		}
	}

	bs, err := json.Marshal(orgsRaw)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return bs, nil
}

func listOrganizations(orgs []*mono_models.Organization) {
	rows := [][]interface{}{}
	if len(orgs) == 0 {
		print.Bold(locale.T("organization_no_orgs"))
		return
	}
	for _, org := range orgs {
		rows = append(rows, []interface{}{org.Name})
	}

	t := gotabulate.Create(rows)
	t.SetHeaders([]string{locale.T("organization_name")})
	t.SetAlign("left")

	print.Line(t.Render("simple"))
}
