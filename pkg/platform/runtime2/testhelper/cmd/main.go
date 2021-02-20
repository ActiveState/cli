package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/testhelper"
	"github.com/go-openapi/strfmt"
)

// This script downloads some test data from real server commits

func main() {
	err := run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

const PerlCommit = strfmt.UUID("bdd07a41-1e71-4042-b666-33c17164c9d9")
const PerlProject = "Perl"

func run() error {
	d := model.NewDefault()
	checkpoint, _, err := d.FetchCheckpointForCommit(PerlCommit)
	if err != nil {
		return fmt.Errorf("Failed to get checkpoint 'perl-order': %w", err)
	}
	testhelper.SaveCheckpoint("perl-order", checkpoint)

	r, err := d.ResolveRecipe(PerlCommit, "ActiveState-CLI", PerlProject)
	if err != nil {
		return fmt.Errorf("Failed to get recipe 'perl-recipe': %w", err)
	}
	testhelper.SaveRecipe("perl-recipe", r)

	be, b, err := d.RequestBuild(*r.RecipeID, PerlCommit, "ActiveState-CLI", PerlProject)
	if err != nil {
		return fmt.Errorf("Failed to get build 'perl-build': %w", err)
	}
	if be != headchef.Completed {
		return errors.New("Expected build to be completed")
	}
	testhelper.SaveBuildResponse("perl-recipe", b)
	return nil
}
