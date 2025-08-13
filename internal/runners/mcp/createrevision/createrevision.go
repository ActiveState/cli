package createrevision

import (
	"encoding/json"
	"fmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

type CreateRevisionRunner struct {
	auth   *authentication.Auth
	output output.Outputer
}

func New(p *primer.Values) *CreateRevisionRunner {
	return &CreateRevisionRunner{
		auth:   p.Auth(),
		output: p.Output(),
	}
}

type Params struct {
	namespace    string
	name         string
	version      string
	dependencies string
	comment      string
}

func NewParams(namespace string, name string, version string, dependencies string, comment string) *Params {
	return &Params{
		namespace:    namespace,
		name:         name,
		version:      version,
		dependencies: dependencies,
		comment:      comment,
	}
}

func (runner *CreateRevisionRunner) Run(params *Params) error {
	// Unmarshal JSON with the new dependency info
	var dependencies []inventory_models.Dependency
	err := json.Unmarshal([]byte(params.dependencies), &dependencies)
	if err != nil {
		return fmt.Errorf("error unmarshaling dependencies, dependency JSON is in wrong format: %w", err)
	}

	// Retrieve ingredient version to access ingredient and version IDs
	ingredient, err := model.GetIngredientByNameAndVersion(params.namespace, params.name, params.version, nil, runner.auth)
	if err != nil {
		return fmt.Errorf("error fetching ingredient: %w", err)
	}

	// Retrieve its latest revision to copy all data - but comment and dependencies - from
	getParams := inventory_operations.NewGetIngredientVersionRevisionsParams()
	getParams.SetIngredientID(*ingredient.IngredientID)
	getParams.SetIngredientVersionID(*ingredient.IngredientVersionID)

	client := inventory.Get(runner.auth)
	revisions, err := client.GetIngredientVersionRevisions(getParams, runner.auth.ClientAuth())
	if err != nil {
		return fmt.Errorf("error getting version revisions: %w", err)
	}
	revision := revisions.Payload.IngredientVersionRevisions[len(revisions.Payload.IngredientVersionRevisions)-1]

	// Prepare new ingredient version revision params to create a new revision
	// This leaves all the attributes untouched, but dependencies and comments
	newParams := inventory_operations.NewAddIngredientVersionRevisionParams()
	newParams.SetIngredientID(*ingredient.IngredientID)
	newParams.SetIngredientVersionID(*ingredient.IngredientVersionID)

	// Extract build script IDs
	var buildScriptIDs []strfmt.UUID
	for _, script := range revision.BuildScripts {
		buildScriptIDs = append(buildScriptIDs, *script.BuildScriptID)
	}

	// Replicate patches
	var patches []*inventory_models.IngredientVersionRevisionCreatePatch
	for _, patch := range revision.Patches {
		patches = append(patches, &inventory_models.IngredientVersionRevisionCreatePatch{
			PatchID:        patch.PatchID,
			SequenceNumber: patch.SequenceNumber,
		})
	}

	// Retrieve and prepare default and override option sets
	optsetParams := inventory_operations.NewGetIngredientVersionIngredientOptionSetsParams()
	optsetParams.SetIngredientID(*ingredient.IngredientID)
	optsetParams.SetIngredientVersionID(*ingredient.IngredientVersionID)

	response, err := client.GetIngredientVersionIngredientOptionSets(optsetParams, runner.auth.ClientAuth())
	if err != nil {
		return fmt.Errorf("error getting optsets: %w", err)
	}

	var default_optsets []strfmt.UUID
	var override_optsets []strfmt.UUID
	for _, optset := range response.Payload.IngredientOptionSetsWithUsageType {
		switch *optset.UsageType {
		case "default":
			default_optsets = append(default_optsets, *optset.IngredientOptionSetID)
		case "override":
			override_optsets = append(override_optsets, *optset.IngredientOptionSetID)
		}
	}

	// Create and set the new revision object from model, setting the reason as manual change
	manual_change := inventory_models.IngredientVersionRevisionCoreReasonManualChange
	new_revision := inventory_models.IngredientVersionRevisionCreate{
		IngredientVersionRevisionCore: inventory_models.IngredientVersionRevisionCore{
			Comment:                      &params.comment,
			ProvidedFeatures:             revision.ProvidedFeatures,
			Reason:                       &manual_change,
			ActivestateLicenseExpression: revision.ActivestateLicenseExpression,
			AuthorPlatformUserID:         revision.AuthorPlatformUserID,
			CamelExtras:                  revision.CamelExtras,
			Dependencies:                 dependencies,
			IsIndemnified:                revision.IsIndemnified,
			IsStableRelease:              revision.IsStableRelease,
			IsStableRevision:             revision.IsStableRevision,
			LicenseManifestURI:           revision.LicenseManifestURI,
			PlatformSourceURI:            revision.PlatformSourceURI,
			ScannerLicenseExpression:     revision.ScannerLicenseExpression,
			SourceChecksum:               revision.SourceChecksum,
			Status:                       revision.Status,
		},
		IngredientVersionRevisionCreateAllOf0: inventory_models.IngredientVersionRevisionCreateAllOf0{
			BuildScripts:                 buildScriptIDs,
			DefaultIngredientOptionSets:  default_optsets,
			IngredientOptionSetOverrides: override_optsets,
			Patches:                      patches,
		},
	}
	newParams.SetIngredientVersionRevision(&new_revision)

	// Create the new revision and output its marshalled string
	newRevision, err := client.AddIngredientVersionRevision(newParams, runner.auth.ClientAuth())
	if err != nil {
		return fmt.Errorf("error creating revision: %w", err)
	}

	marshalledRevision, err := json.Marshal(newRevision)
	if err != nil {
		return fmt.Errorf("error marshalling revision response: %w", err)
	}
	runner.output.Print(string(marshalledRevision))

	return nil
}
