package buildscript

import (
	"encoding/json"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/buildscript/internal/buildexpression"
	"github.com/ActiveState/cli/pkg/buildscript/internal/raw"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

// BuildScript is what we want consuming code to work with. This specifically makes the raw presentation private as no
// consuming code should ever be looking at the raw representation, instead this package should facilitate the use-case
// of the consuming code through convenience methods that are easy to understand and work with.
type BuildScript struct {
	// buildexpression is what we do all our modifications on. We will be planning work to move this to the raw type
	// instead, but for now this is where most of the actual low level modification logic is done.
	buildexpression *buildexpression.BuildExpression
	atTime          *time.Time
}

func New() (*BuildScript, error) {
	expr, err := buildexpression.New()
	if err != nil {
		return nil, errs.Wrap(err, "Could not create empty build expression")
	}
	script, err := unmarshalBuildExpressionTyped(expr, nil)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create empty build expression")
	}
	return script, nil
}

// Unmarshal will parse a buildscript from its presentation on disk
// This needs to unmarshal the ascript representation, and then convert that representation into a build expression
func Unmarshal(data []byte) (*BuildScript, error) {
	raw, err := raw.Unmarshal(data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal buildscript")
	}

	be, err := raw.MarshalBuildExpression()
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build expression from raw")
	}

	expr, err := buildexpression.Unmarshal(be)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	return &BuildScript{expr, raw.AtTime}, nil
}

// UnmarshalBuildExpression will create buildscript using an existing build expression
func UnmarshalBuildExpression(b []byte, atTime *time.Time) (*BuildScript, error) {
	expr, err := buildexpression.Unmarshal(b)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse build expression")
	}

	return unmarshalBuildExpressionTyped(expr, atTime)
}

func unmarshalBuildExpressionTyped(expr *buildexpression.BuildExpression, atTime *time.Time) (*BuildScript, error) {
	// Copy incoming build expression to keep any modifications local.
	var err error
	expr, err = expr.Copy()
	if err != nil {
		return nil, errs.Wrap(err, "Could not copy build expression")
	}

	// Update old expressions that bake in at_time as a timestamp instead of as a variable.
	err = expr.MaybeSetDefaultAtTime(atTime)
	if err != nil {
		return nil, errs.Wrap(err, "Could not set default timestamp")
	}

	return &BuildScript{expr, atTime}, nil
}

// MarshalBuildExpression translates our buildscript into a build expression
// The actual logic for this lives under the MarshalJSON methods below, named that way because that's what the json
// marshaller is expecting to find.
func (b *BuildScript) MarshalBuildExpression() ([]byte, error) {
	bytes, err := json.MarshalIndent(b.buildexpression, "", "  ")
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal build script to build expression")
	}
	return bytes, nil
}

// Requirements returns the requirements in the Buildscript
// It returns an error if the requirements are not found or if they are malformed.
func (b *BuildScript) Requirements() ([]types.Requirement, error) {
	return b.buildexpression.Requirements()
}

// RequirementNotFoundError aliases the buildexpression error type, which can otherwise not be checked as its internal
type RequirementNotFoundError = buildexpression.RequirementNotFoundError

// UpdateRequirement updates the Buildscripts requirements based on the operation and requirement.
func (b *BuildScript) UpdateRequirement(operation types.Operation, requirement types.Requirement) error {
	return b.buildexpression.UpdateRequirement(operation, requirement)
}

func (b *BuildScript) UpdatePlatform(operation types.Operation, platformID strfmt.UUID) error {
	return b.buildexpression.UpdatePlatform(operation, platformID)
}

func (b *BuildScript) AtTime() *time.Time {
	return b.atTime
}

func (b *BuildScript) SetAtTime(t time.Time) {
	b.atTime = &t
}

func (b *BuildScript) SetDefaultAtTime() error {
	return b.buildexpression.SetDefaultAtTime()
}

// MaybeSetDefaultAtTime changes the solve node's "at_time" value to "$at_time" if and only if
// the current value is the given timestamp.
// Buildscripts prefer to use variables for at_time and define them outside the buildscript as
// the expression's commit time.
// While modern buildscripts use variables, older ones bake in the commit time. This function
// exists primarily to update those older buildscripts for use in buildscripts.
func (b *BuildScript) MaybeSetDefaultAtTime(ts *time.Time) error {
	return b.buildexpression.MaybeSetDefaultAtTime(ts)
}

func (b *BuildScript) Marshal() ([]byte, error) {
	raw, err := raw.UnmarshalBuildExpression(b.buildexpression, b.atTime)
	if err != nil {
		return []byte(""), errs.Wrap(err, "Could not unmarshal build expression to raw")
	}
	return raw.Marshal()
}

func (b *BuildScript) Equals(other *BuildScript) (bool, error) {
	// Compare top-level at_time.
	switch {
	case b.atTime != nil && other.atTime != nil && b.atTime.String() != other.atTime.String():
		return false, nil
	case (b.atTime == nil) != (other.atTime == nil):
		return false, nil
	}

	// Compare buildexpression JSON.
	myJson, err := b.MarshalBuildExpression()
	if err != nil {
		return false, errs.New("Unable to marshal this buildscript to JSON: %s", errs.JoinMessage(err))
	}
	otherJson, err := other.MarshalBuildExpression()
	if err != nil {
		return false, errs.New("Unable to marshal other buildscript to JSON: %s", errs.JoinMessage(err))
	}
	return string(myJson) == string(otherJson), nil
}

func (b *BuildScript) Merge(b2 *BuildScript, strategies *mono_models.MergeStrategies) (*BuildScript, error) {
	expr, err := buildexpression.Merge(b.buildexpression, b2.buildexpression, strategies)
	if err != nil {
		return nil, errs.Wrap(err, "Could not merge build expressions")
	}

	atTime := b.atTime
	if b.atTime != nil && b.atTime.After(*b2.atTime) {
		atTime = b2.atTime
	}

	bs, err := unmarshalBuildExpressionTyped(expr, atTime)

	// For now, pick the later of the script AtTimes.
	return bs, nil
}
