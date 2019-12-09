package pkg

import (
	"github.com/go-openapi/strfmt"
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/project"
)

// Package holds package-related data
type Package struct{}

// ListFlags holds the list-related flag values passed through the command line
var ListFlags struct {
	Commit string
}

// ExecuteList lists the current packages in a project
func ExecuteList(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteList")

	proj := project.Get()

	commit, fail := targetedCommit(proj, ListFlags.Commit)
	if fail != nil {
		failures.Handle(fail, "")
		return
	}

	pkgs, fail := makePackages(commit)
	if fail != nil {
		failures.Handle(fail, "")
		return
	}

	printList(pkgs)
}

func targetedCommit(proj *project.Project, commitFlag string) (strfmt.UUID, *failures.Failure) {
	return "", nil
}

func makePackages(commit strfmt.UUID) ([]*Package, *failures.Failure) {
	return nil, nil
}

func printList(pkgs []*Package) {
	print.Info("not json")
}
