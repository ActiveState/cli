package changesummary

import "github.com/ActiveState/cli/pkg/platform/runtime/artifact"

// EmptyChangeSummary is used for runners that do not wish to print a change summary to the user
type EmptyChangeSummary struct{}

// NewEmpty returns a new EmptyChangeSummary
func NewEmpty() *EmptyChangeSummary {
	return &EmptyChangeSummary{}
}

func (c *EmptyChangeSummary) ChangeSummary(artifacts artifact.ArtifactRecipeMap, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) error {
	return nil
}
