package organizations

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/bndr/gotabulate"
	"github.com/spf13/cobra"
)

// Command is the organization command's definition.
var Command = &commands.Command{
	Name:        "organizations",
	Aliases:     []string{"orgs"},
	Description: "organizations_description",
	Run:         Execute,
}

// Execute the organizations command.
func Execute(cmd *cobra.Command, args []string) {
	orgs, fail := model.FetchOrganizations()
	if fail != nil {
		failures.Handle(fail, locale.T("organizations_err"))
		return
	}

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
