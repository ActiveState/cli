package runtime

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/internal/buildlog"
)

// ProgressReportError designates an error in the event handler for reporting progress.
type ProgressReportError struct {
	*errs.WrapperError
}

// buildlog aliases, because buildlog is internal
type ArtifactBuildError = buildlog.ArtifactBuildError
type BuildError = buildlog.BuildError

// ArtifactCachedBuildFailed designates an error due to a build for an artifact that failed and has been cached
type ArtifactCachedBuildFailed struct {
	*errs.WrapperError
	Artifact *buildplan.Artifact
}
