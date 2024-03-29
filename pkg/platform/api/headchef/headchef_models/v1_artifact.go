// Code generated by go-swagger; DO NOT EDIT.

package headchef_models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V1Artifact Artifact (V1)
//
// The result of building a single ingredient is an artifact, which contains the files created by the build.
// swagger:model V1Artifact
type V1Artifact struct {

	// Artifact ID
	// Required: true
	// Format: uuid
	ArtifactID *strfmt.UUID `json:"artifact_id"`

	// Indicates where in the build process the artifact currently is.
	// Required: true
	// Enum: [blocked doomed failed ready running skipped starting succeeded]
	BuildState *string `json:"build_state"`

	// Timestamp for when the artifact was created
	// Required: true
	// Format: date-time
	BuildTimestamp *strfmt.DateTime `json:"build_timestamp"`

	// checksum
	Checksum string `json:"checksum,omitempty"`

	// dependency ids
	DependencyIds []strfmt.UUID `json:"dependency_ids"`

	// The error that happened which caused the artifact to fail to build. Only non-null if 'build_state' is 'failed'.
	Error string `json:"error,omitempty"`

	// Ingredient Version ID
	//
	// Source Ingredient Version ID for the artifact. Null if the artifact was not built directly from an ingredient (i.e. a packager artifact)
	// Format: uuid
	IngredientVersionID strfmt.UUID `json:"ingredient_version_id,omitempty"`

	// URI for the storage location of the artifact's build log. Only artifacts that have finished building with the 'alternative' build engine have artifact logs. For all other cases this field is always null.
	// Format: uri
	LogURI strfmt.URI `json:"log_uri,omitempty"`

	// The MIME type of the file stored at the artifact's URI. Only artifacts built with the 'alternative' build engine have MIME types. For all other build engines this field is always null.
	MimeType string `json:"mime_type,omitempty"`

	// Platform ID for the artifact
	// Required: true
	// Format: uuid
	PlatformID *strfmt.UUID `json:"platform_id"`

	// URI for the storage location of the artifact. Only non-null if 'build_state' is 'succeeded'.
	// Format: uri
	URI strfmt.URI `json:"uri,omitempty"`
}

// Validate validates this v1 artifact
func (m *V1Artifact) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateArtifactID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBuildState(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBuildTimestamp(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDependencyIds(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIngredientVersionID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLogURI(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePlatformID(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateURI(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V1Artifact) validateArtifactID(formats strfmt.Registry) error {

	if err := validate.Required("artifact_id", "body", m.ArtifactID); err != nil {
		return err
	}

	if err := validate.FormatOf("artifact_id", "body", "uuid", m.ArtifactID.String(), formats); err != nil {
		return err
	}

	return nil
}

var v1ArtifactTypeBuildStatePropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["blocked","doomed","failed","ready","running","skipped","starting","succeeded"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		v1ArtifactTypeBuildStatePropEnum = append(v1ArtifactTypeBuildStatePropEnum, v)
	}
}

const (

	// V1ArtifactBuildStateBlocked captures enum value "blocked"
	V1ArtifactBuildStateBlocked string = "blocked"

	// V1ArtifactBuildStateDoomed captures enum value "doomed"
	V1ArtifactBuildStateDoomed string = "doomed"

	// V1ArtifactBuildStateFailed captures enum value "failed"
	V1ArtifactBuildStateFailed string = "failed"

	// V1ArtifactBuildStateReady captures enum value "ready"
	V1ArtifactBuildStateReady string = "ready"

	// V1ArtifactBuildStateRunning captures enum value "running"
	V1ArtifactBuildStateRunning string = "running"

	// V1ArtifactBuildStateSkipped captures enum value "skipped"
	V1ArtifactBuildStateSkipped string = "skipped"

	// V1ArtifactBuildStateStarting captures enum value "starting"
	V1ArtifactBuildStateStarting string = "starting"

	// V1ArtifactBuildStateSucceeded captures enum value "succeeded"
	V1ArtifactBuildStateSucceeded string = "succeeded"
)

// prop value enum
func (m *V1Artifact) validateBuildStateEnum(path, location string, value string) error {
	if err := validate.Enum(path, location, value, v1ArtifactTypeBuildStatePropEnum); err != nil {
		return err
	}
	return nil
}

func (m *V1Artifact) validateBuildState(formats strfmt.Registry) error {

	if err := validate.Required("build_state", "body", m.BuildState); err != nil {
		return err
	}

	// value enum
	if err := m.validateBuildStateEnum("build_state", "body", *m.BuildState); err != nil {
		return err
	}

	return nil
}

func (m *V1Artifact) validateBuildTimestamp(formats strfmt.Registry) error {

	if err := validate.Required("build_timestamp", "body", m.BuildTimestamp); err != nil {
		return err
	}

	if err := validate.FormatOf("build_timestamp", "body", "date-time", m.BuildTimestamp.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1Artifact) validateDependencyIds(formats strfmt.Registry) error {

	if swag.IsZero(m.DependencyIds) { // not required
		return nil
	}

	for i := 0; i < len(m.DependencyIds); i++ {

		if err := validate.FormatOf("dependency_ids"+"."+strconv.Itoa(i), "body", "uuid", m.DependencyIds[i].String(), formats); err != nil {
			return err
		}

	}

	return nil
}

func (m *V1Artifact) validateIngredientVersionID(formats strfmt.Registry) error {

	if swag.IsZero(m.IngredientVersionID) { // not required
		return nil
	}

	if err := validate.FormatOf("ingredient_version_id", "body", "uuid", m.IngredientVersionID.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1Artifact) validateLogURI(formats strfmt.Registry) error {

	if swag.IsZero(m.LogURI) { // not required
		return nil
	}

	if err := validate.FormatOf("log_uri", "body", "uri", m.LogURI.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1Artifact) validatePlatformID(formats strfmt.Registry) error {

	if err := validate.Required("platform_id", "body", m.PlatformID); err != nil {
		return err
	}

	if err := validate.FormatOf("platform_id", "body", "uuid", m.PlatformID.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V1Artifact) validateURI(formats strfmt.Registry) error {

	if swag.IsZero(m.URI) { // not required
		return nil
	}

	if err := validate.FormatOf("uri", "body", "uri", m.URI.String(), formats); err != nil {
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *V1Artifact) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1Artifact) UnmarshalBinary(b []byte) error {
	var res V1Artifact
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
