package buildscript

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildscript/internal/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

// BuildScript is what we want consuming code to work with. This specifically makes the raw
// presentation private as no consuming code should ever be looking at the raw representation.
// Instead this package should facilitate the use-case of the consuming code through convenience
// methods that are easy to understand and work with.
type BuildScript struct {
	raw *raw.Raw
}

func New() (*BuildScript, error) {
	raw, err := raw.New()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create empty build script")
	}
	return &BuildScript{raw}, nil
}

// Unmarshal returns a BuildScript from the given AScript (on-disk format).
func Unmarshal(data []byte) (*BuildScript, error) {
	raw, err := raw.Unmarshal(data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}
	return &BuildScript{raw}, nil
}

// UnmarshalBuildExpression returns a BuildScript constructed from the given build expression in
// JSON format.
// Build scripts and build expressions are almost identical, with the exception of the atTime field.
// Build expressions ALWAYS set at_time to `$at_time`, which refers to the timestamp on the commit,
// while buildscripts encode this timestamp as part of their definition. For this reason we have
// to supply the timestamp as a separate argument.
func UnmarshalBuildExpression(data []byte, atTime *time.Time) (*BuildScript, error) {
	raw, err := raw.UnmarshalBuildExpression(data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}
	raw.AtTime = atTime
	return &BuildScript{raw}, nil
}

func (b *BuildScript) AtTime() *time.Time {
	return b.raw.AtTime
}

func (b *BuildScript) SetAtTime(t time.Time) {
	b.raw.AtTime = &t
}

// Marshal returns this BuildScript in AScript format, suitable for writing to disk.
func (b *BuildScript) Marshal() ([]byte, error) {
	return b.raw.Marshal()
}

// MarshalBuildExpression returns for this BuildScript a build expression in JSON format, suitable
// for sending to the Platform.
func (b *BuildScript) MarshalBuildExpression() ([]byte, error) {
	return b.raw.MarshalBuildExpression()
}

func (b *BuildScript) Requirements() ([]types.Requirement, error) {
	return b.raw.Requirements()
}

type RequirementNotFoundError = raw.RequirementNotFoundError // expose
type PlatformNotFoundError = raw.PlatformNotFoundError       // expose

func (b *BuildScript) UpdateRequirement(operation types.Operation, requirement types.Requirement) error {
	return b.raw.UpdateRequirement(operation, requirement)
}

func (b *BuildScript) Platforms() ([]strfmt.UUID, error) {
	return b.raw.Platforms()
}

func (b *BuildScript) UpdatePlatform(operation types.Operation, platformID strfmt.UUID) error {
	return b.raw.UpdatePlatform(operation, platformID)
}

func (b *BuildScript) Equals(other *BuildScript) (bool, error) {
	myBytes, err := b.Marshal()
	if err != nil {
		return false, errs.New("Unable to marshal this buildscript: %s", errs.JoinMessage(err))
	}
	otherBytes, err := other.Marshal()
	if err != nil {
		return false, errs.New("Unable to marshal other buildscript: %s", errs.JoinMessage(err))
	}
	return string(myBytes) == string(otherBytes), nil
}
