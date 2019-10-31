package activate

import (
	"fmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
)

func createProjectFile(org, project, directory string, commitID *strfmt.UUID) *failures.Failure {
	fail := fileutils.MkdirUnlessExists(directory)
	if fail != nil {
		return fail
	}

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, org, project)
	if commitID != nil {
		projectURL = fmt.Sprintf("%s?commitID=%s", projectURL, commitID.String())
	}

	_, fail = projectfile.Create(projectURL, directory)
	if fail != nil {
		return fail
	}

	return nil
}
