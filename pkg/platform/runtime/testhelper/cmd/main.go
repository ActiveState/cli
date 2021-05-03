package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/testhelper"
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

// Perl5_28CamelProject is the project name in the ActiveState-CLI organization
const Perl5_28CamelProject = "Perl"

// Perl5_28CamelCommit is a test commit for a camel Perl build
const Perl5_28CamelCommit = strfmt.UUID("bdd07a41-1e71-4042-b666-33c17164c9d9")

// Perl5_32AlternativeProject is a project name in the ActiveState-CLI organization for an alternative build
const Perl5_32AlternativeProject = "Perl-5.32.1-Alternative"

// Perl5_32AlternativeBaseCommit only comprises the bare Perl language
const Perl5_32AlternativeBaseCommit = strfmt.UUID("a5075f1a-053f-4cb7-b1fd-e8c09b8371f3")

// Perl5_32AlternativeOnePackageCommit comprises the Perl language and JSON as a specified version
const Perl5_32AlternativeOnePackageCommit = strfmt.UUID("ab02c350-c4ff-4415-a7ad-3de3bbb1c67a")

// Perl5_32AlternativeOnePackageUpdatedCommit comprises the Perl language and JSON at an updated version
const Perl5_32AlternativeOnePackageUpdatedCommit = strfmt.UUID("e2b4f7f1-c878-4e57-8d1a-7bf6cc8c1abc")

// Perl5_32AlternativeOnePackageRemovedCommit only the Perl language after removing the JSON package
const Perl5_32AlternativeOnePackageRemovedCommit = strfmt.UUID("bd940c5e-756b-4929-9517-2ae90e9499e4")

// Perl5_32AlternativeOneBundleCommit comprises the Perl language and one bundle 'Testing'
const Perl5_32AlternativeOneBundleCommit = strfmt.UUID("d0118507-f60e-4602-9355-7196c1aed3ca")

// Perl5_32AlternativeFailedCommit has a package that makes the build fail
const Perl5_32AlternativeFailedCommit = strfmt.UUID("adeabd0f-cf90-4b65-8f0b-924ae15c9338")

func saveResponses(baseName string, commitID strfmt.UUID, projectName string, expectedBuildResult headchef.BuildStatusEnum) error {
	fmt.Printf("Downloading build for %s\n", baseName)
	d := model.NewDefault()

	r, err := d.ResolveRecipe(commitID, "ActiveState-CLI", projectName)
	if err != nil {
		return fmt.Errorf("Failed to get recipe '%s': %w", baseName, err)
	}
	testhelper.SaveRecipe(baseName, r)

	be, b, err := d.RequestBuild(*r.RecipeID, commitID, "ActiveState-CLI", projectName)
	if err != nil && !(expectedBuildResult == headchef.Failed && errors.Is(err, headchef.ErrBuildFailedResp)) {
		return fmt.Errorf("Failed to get build '%s': %w", baseName, err)
	}
	if be != expectedBuildResult {
		return fmt.Errorf("Expected build to be %v", expectedBuildResult)
	}
	testhelper.SaveBuildResponse(baseName, b)
	return nil
}

func run() error {
	if err := saveResponses("perl", Perl5_28CamelCommit, Perl5_28CamelProject, headchef.Completed); err != nil {
		return err
	}
	if err := saveResponses("perl-alternative-base", Perl5_32AlternativeBaseCommit, Perl5_32AlternativeProject, headchef.Completed); err != nil {
		return err
	}
	if err := saveResponses("perl-alternative-one-package", Perl5_32AlternativeOnePackageCommit, Perl5_32AlternativeProject, headchef.Completed); err != nil {
		return err
	}
	if err := saveResponses("perl-alternative-one-update", Perl5_32AlternativeOnePackageUpdatedCommit, Perl5_32AlternativeProject, headchef.Completed); err != nil {
		return err
	}
	if err := saveResponses("perl-alternative-one-removed", Perl5_32AlternativeOnePackageRemovedCommit, Perl5_32AlternativeProject, headchef.Completed); err != nil {
		return err
	}
	if err := saveResponses("perl-alternative-one-bundle", Perl5_32AlternativeOneBundleCommit, Perl5_32AlternativeProject, headchef.Completed); err != nil {
		return err
	}
	if err := saveResponses("perl-alternative-failure", Perl5_32AlternativeFailedCommit, Perl5_32AlternativeProject, headchef.Failed); err != nil {
		return err
	}
	return nil
}
